#!/usr/bin/env bash
set -euo pipefail
export KUBEVISTA_LAB_CONTEXT=kind-kubevista-ci
api=http://127.0.0.1:18080
mkdir -p outputs/incident-lab
k=(kubectl --context "$KUBEVISTA_LAB_CONTEXT")
cleanup(){ [[ -z "${forward_pid:-}" ]] || kill "$forward_pid" 2>/dev/null || true; }
trap cleanup EXIT
bash scripts/incident-lab.sh setup
docker build -t kubevista-api:lab api
docker build -t kubevista-web:lab web
kind load docker-image kubevista-api:lab kubevista-web:lab --name kubevista-ci
helm upgrade --install kubevista platform/apps/kubevista-local --kube-context "$KUBEVISTA_LAB_CONTEXT" -n kubevista-system --create-namespace --set operations.enabled=true --set nodeOperator.enabled=true --wait --timeout 180s
"${k[@]}" -n kubevista-system port-forward svc/kubevista-api 18080:80 >outputs/incident-lab/port-forward.log 2>&1 &
forward_pid=$!
for _ in {1..30}; do curl -fsS "$api/readyz" >/dev/null && break; sleep 1; done
curl -fsS "$api/readyz" >/dev/null
base=(--api "$api" --namespace kubevista-lab --name probe-failure)
for scenario in probe crashloop imagepull scheduling oom; do
  case "$scenario" in probe) code=ProbeFailed;;crashloop) code=CrashLoopBackOff;;imagepull) code=ImagePullBackOff;;scheduling) code=Unschedulable;;oom) code=OOMKilled;;esac
  start=$(date +%s)
  bash scripts/incident-lab.sh "$scenario"
  found=false
  for _ in {1..90}; do
    bin/kubevista diagnose "${base[@]}" >"outputs/incident-lab/$scenario-diagnosis.json"
    if jq -e --arg code "$code" '.diagnoses|any(.code==$code)' "outputs/incident-lab/$scenario-diagnosis.json" >/dev/null; then found=true;break;fi
    sleep 2
  done
  [[ "$found" == true ]] || { "${k[@]}" -n kubevista-lab get pods;exit 1; }
  diagnosed=$(date +%s)
  if [[ "$scenario" == crashloop ]]; then
    bin/kubevista diagnose "${base[@]}" --evidence >outputs/incident-lab/crashloop-enriched.json
    jq -e '.logs|any(.text|contains("injected-crash"))' outputs/incident-lab/crashloop-enriched.json >/dev/null
  fi
  # Kubernetes controllers may update resourceVersion during review; retry a new
  # review, never reuse a consumed or stale plan.
  accepted=false
  for _ in {1..10}; do
    bin/kubevista plan "${base[@]}" --action rollback --reason "Recover from injected $scenario failure" >"outputs/incident-lab/$scenario-plan.json"
    if bin/kubevista execute --api "$api" --plan "outputs/incident-lab/$scenario-plan.json" >"outputs/incident-lab/$scenario-receipt.json"; then accepted=true;break;fi
    sleep 1
  done
  [[ "$accepted" == true ]]
  bin/kubevista verify "${base[@]}" --timeout 180s >"outputs/incident-lab/$scenario-recovery.json"
  end=$(date +%s)
  jq -n --arg scenario "$scenario" --arg code "$code" --argjson diagnosisSeconds "$((diagnosed-start))" --argjson recoverySeconds "$((end-diagnosed))" '{scenario:$scenario,expectedDiagnosis:$code,diagnosisSeconds:$diagnosisSeconds,recoverySeconds:$recoverySeconds,environment:"kind, GitHub Actions; single run"}' >"outputs/incident-lab/$scenario-measurement.json"
  # A reviewed plan is single use, even after recovery.
  if bin/kubevista execute --api "$api" --plan "outputs/incident-lab/$scenario-plan.json" >outputs/incident-lab/replay.json 2>&1; then exit 1;fi
done
# A change after review must invalidate the plan on an actual API server too.
bin/kubevista plan "${base[@]}" --action restart --reason "Verify concurrent change rejection" >outputs/incident-lab/stale-plan.json
"${k[@]}" -n kubevista-lab annotate deployment probe-failure kubevista.dev/concurrent-change="$(date +%s)" --overwrite
if bin/kubevista execute --api "$api" --plan outputs/incident-lab/stale-plan.json >outputs/incident-lab/stale-denied.json 2>&1; then exit 1;fi
grep -q stale_plan outputs/incident-lab/stale-denied.json
if bin/kubevista plan --api "$api" --namespace kube-system --name coredns --action restart --reason "Verify namespace restriction" >outputs/incident-lab/namespace-denied.json 2>&1; then exit 1;fi
if bin/kubevista plan "${base[@]}" --action scale --replicas 99 --reason "Verify replica guardrail" >outputs/incident-lab/replicas-denied.json 2>&1; then exit 1;fi
[[ $("${k[@]}" auth can-i patch nodes --as=system:serviceaccount:kubevista-system:kubevista) == no ]]
bash scripts/node-cordon.sh plan kubevista-ci-control-plane true "Test separate node operator" >outputs/incident-lab/cordon-plan.json
bash scripts/node-cordon.sh execute outputs/incident-lab/cordon-plan.json >outputs/incident-lab/cordon-receipt.json
bash scripts/node-cordon.sh plan kubevista-ci-control-plane false "Restore lab node scheduling" >outputs/incident-lab/uncordon-plan.json
bash scripts/node-cordon.sh execute outputs/incident-lab/uncordon-plan.json >outputs/incident-lab/uncordon-receipt.json
KUBEVISTA_API="$api" bash scripts/temporary-scale.sh kubevista-lab probe-failure 2 5 >outputs/incident-lab/temporary-scale.json
bin/kubevista verify "${base[@]}" --timeout 180s >outputs/incident-lab/final-recovery.json
bin/kubevista history --api "$api" >outputs/incident-lab/audit.json
jq -s '.' outputs/incident-lab/*-measurement.json >outputs/incident-lab/measurements.json
