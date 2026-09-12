#!/usr/bin/env bash
set -euo pipefail
# All mutations are explicitly bound to the dedicated kind context.
scenario="${1:-probe}"
context="${KUBEVISTA_LAB_CONTEXT:-kind-kubevista-lab}"
case "$context" in kind-kubevista-lab|kind-kubevista-ci) ;; *) echo "Expected a dedicated KubeVista kind context" >&2; exit 1;; esac
k=(kubectl --context "$context" -n kubevista-lab)
case "$scenario" in
  setup)
    kubectl --context "$context" apply -f platform/examples/incident-lab.yaml
    "${k[@]}" patch deployment probe-failure --type=json -p='[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet/path","value":"/"}]'
    "${k[@]}" rollout status deployment/probe-failure --timeout=180s
    ;;
  probe) "${k[@]}" patch deployment probe-failure --type=json -p='[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet/path","value":"/broken"}]' ;;
  crashloop) "${k[@]}" patch deployment probe-failure --type=json -p='[{"op":"add","path":"/spec/template/spec/containers/0/command","value":["/bin/sh","-c","exit 1"]}]' ;;
  imagepull) "${k[@]}" set image deployment/probe-failure web=nginx:kubevista-nonexistent-image ;;
  scheduling) "${k[@]}" patch deployment probe-failure --type=json -p='[{"op":"replace","path":"/spec/template/spec/containers/0/resources/requests/cpu","value":"1000"},{"op":"replace","path":"/spec/template/spec/containers/0/resources/limits/cpu","value":"1000"}]' ;;
  oom) "${k[@]}" patch deployment probe-failure --type=json -p='[{"op":"add","path":"/spec/template/spec/containers/0/command","value":["/bin/sh","-c","head -c 134217728 /dev/zero > /dev/shm/fill; sleep 3600"]},{"op":"add","path":"/spec/template/spec/volumes","value":[{"name":"memory","emptyDir":{"medium":"Memory"}}]},{"op":"add","path":"/spec/template/spec/containers/0/volumeMounts","value":[{"name":"memory","mountPath":"/dev/shm"}]}]' ;;
  *) echo "Use setup, probe, crashloop, imagepull, scheduling, or oom" >&2; exit 1 ;;
esac
