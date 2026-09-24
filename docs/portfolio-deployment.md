# Portfolio test deployment

This is a prepared deployment, not evidence of a running website in EKS.
The public website stays on its current hosting. There is no Ingress, public
load balancer, DNS change, or automatic Argo CD registration in this chart.

## Build a selected revision

Use a clean checkout of the chosen portfolio commit. Do not include unrelated
local video experiments. From the KubeVista repository:

```sh
docker buildx build --platform linux/amd64 \
  -f platform/apps/portfolio/Dockerfile \
  --build-context portfolio-config=platform/apps/portfolio \
  -t REGISTRY/portfolio:COMMIT --push /absolute/path/to/clean/portfolio
```

Create or select the registry and its publishing identity before pushing.
Record the resulting immutable digest. Container build and runtime verification
are still required; Helm rendering alone does not prove the image works.

## Deploy only after confirming the target

Authenticate with `aws sso login --profile kubevista`. Confirm the cluster
exists and is healthy before applying anything. Recreating a deleted EKS
cluster needs a separate reviewed Terraform plan and cost approval.

```sh
helm upgrade --install portfolio platform/apps/portfolio \
  --kube-context CONFIRMED_CONTEXT --namespace portfolio --create-namespace \
  --set-string image=REGISTRY/portfolio@sha256:DIGEST --wait --timeout 180s
kubectl --context CONFIRMED_CONTEXT -n portfolio port-forward service/portfolio 8088:80
```

Open http://127.0.0.1:8088/ and check the home page, privacy page, videos, images,
resume download, and byte-range video responses. The two replicas have health
probes, resource limits, a read-only root filesystem, and no Kubernetes API
token. NetworkPolicy denies ingress/egress when enforced by the cluster CNI;
port-forward uses the Kubernetes API rather than public Service ingress.
No outbound server traffic is needed for static files. Browser requests to
external websites still originate on the visitor's device.

KubeVista's live inventory can discover the portfolio pods and Service if its
inventory permissions and workload limit include them. Demo data will not show
a real deployment. Keep write access disabled for the portfolio namespace until
explicitly approved; testing rollback should not target the public website.

## Public rollout later

Choose a test hostname, TLS certificate, ingress controller and allowed ingress
sources before adding an Ingress and adjusting the deny policy. Verify the test
hostname first. Move www.andy-wu.com only with explicit approval and a DNS rollback
plan. For this video-heavy static site, CDN delivery remains useful even if the
origin runs in Kubernetes.

To remove the test workload, uninstall the `portfolio` Helm release in the same
confirmed context and namespace. This does not tear down the cluster or registry.
