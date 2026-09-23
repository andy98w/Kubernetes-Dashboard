package kubernetes

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// This suite starts only its own loopback API server and etcd. It never reads
// kubeconfig and cannot accidentally run against a user's AWS context.
func localControlPlane(t *testing.T) kubernetes.Interface {
	t.Helper()
	assets := os.Getenv("KUBEVISTA_TEST_ASSETS")
	if assets == "" {
		t.Skip("set KUBEVISTA_TEST_ASSETS to envtest binaries for real API tests")
	}
	dir := t.TempDir()
	port := func() int {
		l, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		defer l.Close()
		return l.Addr().(*net.TCPAddr).Port
	}
	etcdPort, peerPort, apiPort := port(), port(), port()
	start := func(binary string, args ...string) {
		log, e := os.Create(filepath.Join(dir, binary+".log"))
		if e != nil {
			t.Fatal(e)
		}
		cmd := exec.Command(filepath.Join(assets, binary), args...)
		cmd.Stdout, cmd.Stderr = log, log
		if e = cmd.Start(); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			_ = log.Close()
			if t.Failed() {
				b, _ := os.ReadFile(log.Name())
				if len(b) > 6000 {
					b = b[len(b)-6000:]
				}
				t.Log(string(b))
			}
		})
	}
	etcdURL := fmt.Sprintf("http://127.0.0.1:%d", etcdPort)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", peerPort)
	start("etcd", "--data-dir="+filepath.Join(dir, "etcd"), "--listen-client-urls="+etcdURL, "--advertise-client-urls="+etcdURL, "--listen-peer-urls="+peerURL, "--initial-advertise-peer-urls="+peerURL, "--initial-cluster=default="+peerURL)
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "kubevista-test", Organization: []string{"system:masters"}}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if e = os.WriteFile(certFile, certPEM, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(keyFile, keyPEM, 0600); e != nil {
		t.Fatal(e)
	}
	start("kube-apiserver", "--etcd-servers="+etcdURL, fmt.Sprintf("--secure-port=%d", apiPort), "--bind-address=127.0.0.1", "--advertise-address=127.0.0.1", "--tls-cert-file="+certFile, "--tls-private-key-file="+keyFile, "--client-ca-file="+certFile, "--service-account-key-file="+keyFile, "--service-account-signing-key-file="+keyFile, "--service-account-issuer=https://kubevista.test", "--authorization-mode=RBAC", "--service-cluster-ip-range=10.0.0.0/24", "--disable-admission-plugins=ServiceAccount")
	client, e := kubernetes.NewForConfig(&rest.Config{Host: fmt.Sprintf("https://127.0.0.1:%d", apiPort), TLSClientConfig: rest.TLSClientConfig{CAData: certPEM, CertData: certPEM, KeyData: keyPEM}, Timeout: 3 * time.Second, QPS: 100, Burst: 200})
	if e != nil {
		t.Fatal(e)
	}
	eventually(t, 40*time.Second, func() bool {
		_, e := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
		return e == nil
	})
	return client
}
func eventually(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("condition did not converge")
}

func TestDurableRealAPI(t *testing.T) {
	api := localControlPlane(t)
	ctx := context.Background()
	for _, ns := range []string{"durable-test", "durable-store"} {
		if _, e := api.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{}); e != nil {
			t.Fatal(e)
		}
	}
	makeClient := func(id string) *Client {
		c := NewClientWithOperations(api, "local", true, []string{"durable-test"}, 1, 6)
		c.enableDurable("durable-store", id, true)
		return c
	}
	a, b := makeClient("a"), makeClient("b")
	newPlan := func(t *testing.T, name, action string) (OperationPlan, OperationRequest) {
		t.Helper()
		replicas := int32(2)
		_, e := api.AppsV1().Deployments("durable-test").Create(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: appsv1.DeploymentSpec{Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "nginx:1.27"}}}}}}, metav1.CreateOptions{})
		if e != nil {
			t.Fatal(e)
		}
		request := OperationRequest{Action: action, Namespace: "durable-test", Kind: "Deployment", Name: name, Reason: "test failure recovery"}
		if action == OperationScale {
			desired := int32(3)
			request.Replicas = &desired
		}
		plan, e := a.PlanOperation(ctx, request)
		if e != nil {
			t.Fatal(e)
		}
		request.PlanID, request.ExpectedResourceVersion = plan.ID, plan.ResourceVersion
		return plan, request
	}
	expireClaim := func(id string) {
		cm, op, e := a.readDurable(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		op.ClaimUntil = time.Now().Add(-time.Minute)
		if e = a.updateDurable(ctx, cm, op); e != nil {
			t.Fatal(e)
		}
	}
	state := func(id string) string {
		_, op, e := a.readDurable(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		return op.State
	}
	t.Run("approval survives restart and concurrent replay", func(t *testing.T) {
		p, r := newPlan(t, "approval", OperationScale)
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _, _ = b.ExecuteOperation(ctx, r, "operator") }()
		}
		wg.Wait()
		if state(p.ID) != "Approved" {
			t.Fatal("not approved")
		}
		fresh := makeClient("restarted")
		if _, e := fresh.ExecuteOperation(ctx, r, "operator"); e != nil {
			t.Fatal(e)
		}
		_, op, _ := fresh.readDurable(ctx, p.ID)
		if len(op.History) != 2 {
			t.Fatal("duplicate approval transition")
		}
		altered := r
		altered.Reason = "different reason"
		if _, e := fresh.ExecuteOperation(ctx, altered, "operator"); e == nil {
			t.Fatal("accepted changed request")
		}
	})
	t.Run("crash after apply does not repeat rollout", func(t *testing.T) {
		p, r := newPlan(t, "crash", OperationRestart)
		if _, e := a.ExecuteOperation(ctx, r, "operator"); e != nil {
			t.Fatal(e)
		}
		a.durable.afterApply = func() error { return errors.New("injected process exit after Kubernetes accepted patch") }
		defer func() { a.durable.afterApply = nil }()
		if e := a.ReconcileOperation(ctx, p.ID); e == nil {
			t.Fatal("injection did not fire")
		}
		first, _ := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if state(p.ID) != "Executing" {
			t.Fatal("receipt should remain ambiguous")
		}
		expireClaim(p.ID)
		if e := b.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		second, _ := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if second.Generation != first.Generation || second.ResourceVersion != first.ResourceVersion || state(p.ID) != "Verifying" {
			t.Fatal("repeated effect or missing recovery")
		}
		// No kubelet/controller runs in envtest: explicitly supply observed rollout status.
		second.Status = appsv1.DeploymentStatus{ObservedGeneration: second.Generation, Replicas: 2, UpdatedReplicas: 2, ReadyReplicas: 2, AvailableReplicas: 2}
		if _, e := api.AppsV1().Deployments(r.Namespace).UpdateStatus(ctx, second, metav1.UpdateOptions{}); e != nil {
			t.Fatal(e)
		}
		if e := b.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		if state(p.ID) != "Succeeded" {
			t.Fatal("recovery not recorded")
		}
	})
	t.Run("stale claim and stale target are fenced", func(t *testing.T) {
		p, r := newPlan(t, "fencing", OperationRestart)
		_, _ = a.ExecuteOperation(ctx, r, "operator")
		oldCM, oldOp, _ := a.readDurable(ctx, p.ID)
		if e := a.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		expireClaim(p.ID)
		if e := b.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		transition(&oldOp, "Succeeded", "stale writer")
		if e := a.updateDurable(ctx, oldCM, oldOp); !apierrors.IsConflict(e) {
			t.Fatalf("stale receipt not rejected: %v", e)
		}
		if _, e := a.patchDurable(ctx, oldOp, false); e == nil {
			t.Fatal("stale target patch accepted")
		}
		_, current, _ := a.readDurable(ctx, p.ID)
		if current.Epoch < 2 {
			t.Fatal("claim epoch did not advance")
		}
	})
	t.Run("drift is not rebased", func(t *testing.T) {
		p, r := newPlan(t, "drift", OperationScale)
		_, _ = a.ExecuteOperation(ctx, r, "operator")
		d, _ := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		d.Labels = map[string]string{"external": "changed"}
		_, _ = api.AppsV1().Deployments(r.Namespace).Update(ctx, d, metav1.UpdateOptions{})
		if e := a.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		if state(p.ID) != "NeedsReview" {
			t.Fatal("silently rebased")
		}
	})
	t.Run("expired plan cannot execute", func(t *testing.T) {
		p, r := newPlan(t, "expired", OperationRestart)
		cm, op, _ := a.readDurable(ctx, p.ID)
		op.Plan.CreatedAt = time.Now().Add(-6 * time.Minute)
		if e := a.updateDurable(ctx, cm, op); e != nil {
			t.Fatal(e)
		}
		if _, e := b.ExecuteOperation(ctx, r, "operator"); e == nil {
			t.Fatal("expired approval accepted")
		}
		if state(p.ID) != "Expired" {
			t.Fatal("expiry not persisted")
		}
	})
	t.Run("unapproved operation cannot write", func(t *testing.T) {
		p, r := newPlan(t, "unapproved", OperationRestart)
		if e := b.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		d, e := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if e != nil || d.ResourceVersion != p.ResourceVersion || state(p.ID) != "Planned" {
			t.Fatal("unapproved operation changed target")
		}
	})
	t.Run("competing plans do not overwrite each other", func(t *testing.T) {
		p, r := newPlan(t, "competing", OperationScale)
		other := r
		other.PlanID, other.ExpectedResourceVersion = "", ""
		desired := int32(4)
		other.Replicas = &desired
		p2, e := b.PlanOperation(ctx, other)
		if e != nil {
			t.Fatal(e)
		}
		other.PlanID, other.ExpectedResourceVersion = p2.ID, p2.ResourceVersion
		if _, e = a.ExecuteOperation(ctx, r, "operator"); e != nil {
			t.Fatal(e)
		}
		if _, e = b.ExecuteOperation(ctx, other, "operator"); e != nil {
			t.Fatal(e)
		}
		if e = a.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		if e = b.ReconcileOperation(ctx, p2.ID); e != nil {
			t.Fatal(e)
		}
		if state(p2.ID) != "NeedsReview" {
			t.Fatal("second plan overwrote first")
		}
	})
	t.Run("replacement UID is never modified", func(t *testing.T) {
		p, r := newPlan(t, "replacement", OperationScale)
		if _, e := a.ExecuteOperation(ctx, r, "operator"); e != nil {
			t.Fatal(e)
		}
		d, _ := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if e := api.AppsV1().Deployments(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{}); e != nil {
			t.Fatal(e)
		}
		d.ObjectMeta = metav1.ObjectMeta{Name: r.Name}
		if _, e := api.AppsV1().Deployments(r.Namespace).Create(ctx, d, metav1.CreateOptions{}); e != nil {
			t.Fatal(e)
		}
		if e := b.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		if state(p.ID) != "NeedsReview" {
			t.Fatal("replacement accepted")
		}
	})
	t.Run("rollback keeps the reviewed template", func(t *testing.T) {
		_, r := newPlan(t, "rollback", OperationRestart)
		d, _ := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		d.Annotations = map[string]string{"deployment.kubernetes.io/revision": "2"}
		d, e := api.AppsV1().Deployments(r.Namespace).Update(ctx, d, metav1.UpdateOptions{})
		if e != nil {
			t.Fatal(e)
		}
		previous := d.Spec.Template.DeepCopy()
		previous.Spec.Containers[0].Image = "nginx:1.26"
		rs, e := api.AppsV1().ReplicaSets(r.Namespace).Create(ctx, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rollback-old", Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"}, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(d, appsv1.SchemeGroupVersion.WithKind("Deployment"))}}, Spec: appsv1.ReplicaSetSpec{Selector: d.Spec.Selector, Template: *previous}}, metav1.CreateOptions{})
		if e != nil {
			t.Fatal(e)
		}
		r.Action, r.PlanID, r.ExpectedResourceVersion = OperationRollback, "", ""
		p, e := a.PlanOperation(ctx, r)
		if e != nil {
			t.Fatal(e)
		}
		r.PlanID, r.ExpectedResourceVersion = p.ID, p.ResourceVersion
		// History is mutable, but execution must not silently select a new template.
		rs.Spec.Template.Spec.Containers[0].Image = "nginx:1.25"
		if _, e = api.AppsV1().ReplicaSets(r.Namespace).Update(ctx, rs, metav1.UpdateOptions{}); e != nil {
			t.Fatal(e)
		}
		if _, e = b.ExecuteOperation(ctx, r, "operator"); e != nil {
			t.Fatal(e)
		}
		if e = b.ReconcileOperation(ctx, p.ID); e != nil {
			t.Fatal(e)
		}
		d, _ = api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if d.Spec.Template.Spec.Containers[0].Image != "nginx:1.26" {
			t.Fatal("rollback intent changed after approval")
		}
	})
	t.Run("watch queue processes two active controllers", func(t *testing.T) {
		run, cancel := context.WithCancel(ctx)
		defer cancel()
		done := make(chan error, 2)
		c, d := makeClient("worker-c"), makeClient("worker-d")
		go func() { done <- c.RunOperationController(run) }()
		go func() { done <- d.RunOperationController(run) }()
		p, r := newPlan(t, "watched", OperationRestart)
		_, _ = a.ExecuteOperation(ctx, r, "operator")
		eventually(t, 12*time.Second, func() bool { return state(p.ID) == "Verifying" })
		target, _ := api.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if target.Generation != 2 {
			t.Fatalf("duplicate rollout: %d", target.Generation)
		}
		cancel()
		<-done
		<-done
	})
	t.Run("leader takeover", func(t *testing.T) {
		c, d := makeClient("leader-c"), makeClient("leader-d")
		c.durable.activeActive = false
		d.durable.activeActive = false
		first, stopFirst := context.WithCancel(ctx)
		defer stopFirst()
		second, stopSecond := context.WithCancel(ctx)
		defer stopSecond()
		done := make(chan error, 2)
		go func() { done <- c.RunOperationController(first) }()
		eventually(t, 10*time.Second, func() bool {
			lease, e := api.CoordinationV1().Leases("durable-store").Get(ctx, "kubevista-operation-controller", metav1.GetOptions{})
			return e == nil && lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == c.durable.identity
		})
		go func() { done <- d.RunOperationController(second) }()
		stopFirst()
		eventually(t, 30*time.Second, func() bool {
			lease, e := api.CoordinationV1().Leases("durable-store").Get(ctx, "kubevista-operation-controller", metav1.GetOptions{})
			return e == nil && lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == d.durable.identity
		})
		p, r := newPlan(t, "takeover", OperationScale)
		_, _ = b.ExecuteOperation(ctx, r, "operator")
		eventually(t, 12*time.Second, func() bool { return state(p.ID) == "Verifying" })
		stopSecond()
		<-done
		<-done
	})
}
