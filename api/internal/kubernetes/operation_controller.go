package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/workqueue"
)

// RunOperationController runs with the API process. Every replica may approve
// plans; only the elected replica runs workers by default.
func (c *Client) RunOperationController(ctx context.Context) error {
	if c.durable == nil {
		return nil
	}
	if c.durable.activeActive {
		return c.runOperationWorkers(ctx)
	}
	lock := &resourcelock.LeaseLock{LeaseMeta: metav1.ObjectMeta{Name: "kubevista-operation-controller", Namespace: c.durable.namespace}, Client: c.client.CoordinationV1(), LockConfig: resourcelock.ResourceLockConfig{Identity: c.durable.identity}}
	electionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	started := make(chan context.Context, 1)
	stopped := make(chan struct{})
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: lock, LeaseDuration: 15 * time.Second, RenewDeadline: 10 * time.Second, RetryPeriod: 2 * time.Second,
		// Never release early while a canceled HTTP request may still be in flight.
		ReleaseOnCancel: false,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderCtx context.Context) {
				started <- leaderCtx
			},
			OnStoppedLeading: func() { slog.Info("operation leadership ended", "controller", c.durable.identity) },
		},
	})
	if err != nil {
		return err
	}
	go func() { defer close(stopped); elector.Run(electionCtx) }()
	select {
	case leaderCtx := <-started:
		// Own worker lifetime here, not inside the asynchronous election callback.
		// Leadership loss cancels leaderCtx; a new process gets a new incarnation.
		err = c.runOperationWorkers(leaderCtx)
		cancel()
		<-stopped
	case <-stopped:
	}
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("operation leadership lost; restart controller")
}

func (c *Client) runOperationWorkers(ctx context.Context) error {
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
	defer queue.ShutDown()
	factory := informers.NewSharedInformerFactoryWithOptions(c.client, 30*time.Second, informers.WithNamespace(c.durable.namespace), informers.WithTweakListOptions(func(o *metav1.ListOptions) { o.LabelSelector = operationLabel + "=true" }))
	operations := factory.Core().V1().ConfigMaps().Informer()
	enqueue := func(obj any) {
		if cm, ok := obj.(*corev1.ConfigMap); ok {
			op, err := decodeDurable(cm)
			if err == nil && !terminal(op.State) {
				queue.Add(cm.Name)
			}
		}
	}
	_, err := operations.AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: enqueue, UpdateFunc: func(old, new any) {
		before, e1 := decodeDurable(old.(*corev1.ConfigMap))
		after, e2 := decodeDurable(new.(*corev1.ConfigMap))
		// Claim renewals must not trigger a hot reconcile loop.
		if e1 != nil || e2 != nil || before.State != after.State {
			enqueue(new)
		}
	}})
	if err != nil {
		return err
	}
	indexName := "target"
	if err = operations.AddIndexers(cache.Indexers{indexName: func(obj any) ([]string, error) {
		op, e := decodeDurable(obj.(*corev1.ConfigMap))
		if e != nil {
			return nil, nil
		}
		return []string{op.Request.Namespace + "/" + op.Request.Name}, nil
	}}); err != nil {
		return err
	}
	synced := []cache.InformerSynced{operations.HasSynced}
	factory.Start(ctx.Done())
	for namespace := range c.operations.namespaces {
		deployments := informers.NewSharedInformerFactoryWithOptions(c.client, 30*time.Second, informers.WithNamespace(namespace))
		informer := deployments.Apps().V1().Deployments().Informer()
		changed := func(obj any) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			if d, ok := obj.(*appsv1.Deployment); ok {
				records, e := operations.GetIndexer().ByIndex(indexName, d.Namespace+"/"+d.Name)
				if e == nil {
					for _, record := range records {
						enqueue(record)
					}
				}
			}
		}
		if _, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: changed, UpdateFunc: func(_, next any) { changed(next) }, DeleteFunc: changed}); err != nil {
			return err
		}
		synced = append(synced, informer.HasSynced)
		deployments.Start(ctx.Done())
	}
	if !cache.WaitForCacheSync(ctx.Done(), synced...) {
		return fmt.Errorf("operation informer cache sync interrupted")
	}
	go func() { <-ctx.Done(); queue.ShutDown() }()
	for {
		id, shutdown := queue.Get()
		if shutdown {
			return nil
		}
		func() {
			defer queue.Done(id)
			requestCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			if err := c.ReconcileOperation(requestCtx, id); err != nil {
				if ctx.Err() != nil {
					return
				}
				queue.AddRateLimited(id)
				slog.Warn("operation reconcile will retry", "operation", id, "error", err)
				return
			}
			queue.Forget(id)
			_, op, err := c.readDurable(requestCtx, id)
			if err == nil && !terminal(op.State) {
				queue.AddAfter(id, 3*time.Second)
			}
		}()
	}
}
