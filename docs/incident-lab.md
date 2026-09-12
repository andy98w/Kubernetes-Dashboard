# Diagnose a failed readiness probe

For the full stack, guarded rollback, CLI, five scenarios, and measurement
workflow, use [the incident-response guide](incident-response.md). The exercise
below demonstrates a manual source correction.

This exercise uses a disposable local cluster. It does not provision AWS resources.
Install kind, Docker, and kubectl first, then create a dedicated cluster:

```sh
kind create cluster --name kubevista-lab
kubectl --context kind-kubevista-lab apply -f platform/examples/incident-lab.yaml
```

Run the API against this context and start the web frontend:

```sh
kubectl config use-context kind-kubevista-lab
make run-api
# In another terminal:
make run-web
```

Keep demo mode disabled for both processes. Open Workloads, select the
kubevista-lab namespace, and open probe-failure. After Kubernetes records the
warning, Diagnosis should show ProbeFailed with the actual HTTP probe failure.
The pod can be Running while remaining unready.

The diagnosis suggests checking the path. Restarting the Deployment would retain
the broken path, so correct its source configuration:

```sh
kubectl --context kind-kubevista-lab -n kubevista-lab patch deployment probe-failure --type=json -p='[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet/path","value":"/"}]'
kubectl --context kind-kubevista-lab -n kubevista-lab rollout status deployment/probe-failure --timeout=120s
```

Close and reopen the workload details to fetch a fresh observation. Verify the
new pod is ready. Warning events may outlive the failure; diagnosis text explicitly
distinguishes retained probe events and previous OOM terminations from current state.
Record the time to identify the path and the time to recover if comparing workflows.

Cleanup removes this dedicated local cluster:

```sh
kind delete cluster --name kubevista-lab
```

## Current limits

Diagnoses are deterministic symptom checks over pod status and correlated events:
crash loops, image pull failures, container configuration failures, OOM terminations,
scheduling failures, and probe warnings. They do not infer root causes, inspect
logs unless requested through Load logs & metrics. Namespace event collection
correlates owned resources; Kubernetes retention can still remove old events.
Guarded rollback now restores a reviewed previous template, but restart and scale
do not fix a bad probe. Durable incident history remains future work.
