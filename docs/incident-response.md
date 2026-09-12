# Incident response prototype

KubeVista now follows a failure from observation to a reviewed action and a fresh
recovery check. This work extends the August EKS demo; the new features have not
been exercised on that retired AWS environment.

## Try it in the browser

```sh
cd web
npm ci
VITE_DATA_MODE=demo npm run dev
```

Open Workloads. The incident lab supports probe failures, crash loops, missing
images, OOM kills, and scheduling failures. Inject a scenario and open
probe-failure. Review a rollback, enter a reason, run the simulation, and select
Check recovery. Restarting a bad probe leaves the simulated failure in place.
The lab data is synthetic, separate from the recorded August test.

## Run the actual stack locally

Requires Docker, kind, kubectl, Helm, Go, and jq. The local chart has one API
replica, one web replica, no AWS dependencies, and no monitoring CRD dependency.

```sh
kind create cluster --name kubevista-lab --image kindest/node:v1.34.0
bash scripts/incident-lab.sh setup
docker build -t kubevista-api:lab api
docker build -t kubevista-web:lab web
kind load docker-image kubevista-api:lab kubevista-web:lab --name kubevista-lab
helm upgrade --install kubevista platform/apps/kubevista-local \
  --kube-context kind-kubevista-lab -n kubevista-system --create-namespace \
  --set operations.enabled=true --wait
kubectl --context kind-kubevista-lab -n kubevista-system port-forward svc/kubevista-web 8088:80
```

Open http://127.0.0.1:8088. This evaluation chart uses development identity and
ClusterIP services: access it through localhost port forwarding. It is not a
public authenticated installation. Operations default off; the command above
enables Deployment writes only in kubevista-lab. EKS authentication and network
boundaries remain in the separate production chart.

## CLI

```sh
make build-cli
kubectl --context kind-kubevista-lab -n kubevista-system port-forward svc/kubevista-api 8080:80
# In another terminal:
make demo-incident SCENARIO=probe
bin/kubevista diagnose --namespace kubevista-lab --name probe-failure
bin/kubevista diagnose --namespace kubevista-lab --name probe-failure --evidence
bin/kubevista plan --namespace kubevista-lab --name probe-failure \
  --action rollback --reason "Restore the previous readiness probe" > /tmp/kubevista-review.json
# Read the review JSON before executing:
bin/kubevista execute --plan /tmp/kubevista-review.json
bin/kubevista verify --namespace kubevista-lab --name probe-failure --timeout 180s
bin/kubevista history
```

The CLI calls the same API as the browser. JSON output supports scripts. Use
--api to select a different endpoint. An existing authenticated ALB session
cookie can be supplied with KUBEVISTA_SESSION_COOKIE; the CLI does not manufacture
operator identity. HTTP errors and recovery timeouts exit nonzero.

## Recovery rules

Rollback selects the highest owned ReplicaSet revision below the current one.
It pins a copy of that template during review, removes pod-template-hash, and
lists changed field paths without returning environment variable values. The
previous revision is not assumed to be healthy. Templates can change commands,
probes, resources, or service accounts; review the fields and source revision.

The existing five-minute, single-use plan, namespace check, dry run, and replica
limits also apply. Restart and rollback patches carry resource-version
preconditions so a concurrent change cannot be overwritten. Acceptance means
the API accepted the mutation. Recovery additionally requires the controller to
observe the current generation, all desired replicas updated and available,
and no old replicas or unavailable replicas. Application-level success remains
separate from Deployment availability.

GitOps can overwrite an imperative action. Update or revert the source manifest
after an incident; this prototype does not pause Argo CD.

## Other runbooks

- Retry a rollout with the existing guarded restart action.
- Scale using the same CLI plan/execute flow with --action scale --replicas N.
- Temporary scale: `bash scripts/temporary-scale.sh kubevista-lab probe-failure 2 30`.
  It requests both scale and restoration through the API, retains review files,
  and restores only if UID, generation, and replica count still match. This is a
  foreground prototype, not a durable scheduler. Host failure or SIGKILL prevents
  restoration; failed or skipped restoration exits nonzero and requires review.
- Node cordon is a separate local runbook: enable nodeOperator.enabled in the
  local chart, then `bash scripts/node-cordon.sh plan NODE true "incident reason"`.
  Save the JSON and pass it to `execute FILE`. Use false to review an uncordon.
  This script is restricted to named KubeVista kind contexts and uses a separate
  service account via impersonation. The caller needs impersonation permission;
  the web API service account never receives node patch permission. Plans expire
  after five minutes and patches test UID and resourceVersion. Cordon does not
  drain pods. Receipts go to stdout for capture.

## Evidence and measurements

Workload evidence follows Deployment → owned ReplicaSets → owned pods, including
label expressions. The timeline combines controller conditions, pod transitions,
namespace events, and recent accepted operator receipts. Missing sources are
reported; retained warnings and previous OOM terminations are not proof that the
current application is failing.

Load logs & metrics retrieves at most three excerpts, 30 lines and 8 KiB each,
with one-second log request deadlines. Prometheus is opt-in through
KUBEVISTA_PROMETHEUS_URL (prometheusURL in the local chart). The prototype queries
http_requests_total with status labels and http_request_duration_seconds_bucket,
both labeled namespace and deployment. Applications must expose those metrics;
missing results are unavailable, never zero. Samples are observations, not causal
proof. Log content is displayed as text and should be shared according to the
application's data handling rules.

The Incident lab GitHub workflow creates a disposable kind cluster and measures
time to diagnosis and recovery for five injected failures. It also checks plan
replay, replica/namespace denial, separate node permissions, cordon/uncordon, and
temporary scaling. JSON observations and measurements are uploaded as
incident-lab-evidence. These are single-run lab measurements, not a production SLO.
Backend benchmarks measure diagnosis CPU/allocation cost without Kubernetes I/O:

```sh
cd api
go test ./internal/kubernetes -run '^$' -bench Diagnose -benchmem
```

## Prototype boundaries

Plans and the last 50 operation receipts live in one API process. Use one replica
for this evaluation. Restart loses them; multiple replicas need shared plan/audit
storage before production writes. Timelines are rebuilt from retained Kubernetes
objects and receipts, so they are not durable incident archives. At most 20
degraded Deployments are included in the incident overview; per-workload views
remain available. The local chart is a repository chart using locally built
images, not a published public Helm repository.

Cleanup: `kind delete cluster --name kubevista-lab`.
