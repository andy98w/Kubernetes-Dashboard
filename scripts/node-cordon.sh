#!/usr/bin/env bash
set -euo pipefail
# A separate node operator identity; no cluster-wide writes in the web API.
mode="${1:-}"
context="${KUBEVISTA_LAB_CONTEXT:-kind-kubevista-lab}"
case "$context" in kind-kubevista-lab|kind-kubevista-ci) ;; *) echo "Prototype is limited to KubeVista kind labs" >&2; exit 1;; esac
k=(kubectl --context "$context" --as=system:serviceaccount:kubevista-system:kubevista-node-operator)
if [[ "$mode" == plan ]]; then
  node="${2:?node required}"; desired="${3:-true}"; reason="${4:?reason required}"
  [[ "$desired" == true || "$desired" == false ]] || exit 1
  [[ ${#reason} -ge 8 ]] || exit 1
  obj=$("${k[@]}" get node "$node" -o json)
  plan=$(jq -n --arg context "$context" --arg node "$node" --arg reason "$reason" --arg rv "$(jq -r .metadata.resourceVersion <<<"$obj")" --arg uid "$(jq -r .metadata.uid <<<"$obj")" --argjson desired "$desired" --argjson created "$(date +%s)" '{context:$context,node:$node,reason:$reason,resourceVersion:$rv,uid:$uid,unschedulable:$desired,createdAt:$created}')
elif [[ "$mode" == execute ]]; then
  plan=$(jq -c . "${2:?plan file required}")
  jq -e --arg context "$context" --argjson now "$(date +%s)" '.context==$context and (.createdAt <= $now) and ($now-.createdAt < 300) and (.unschedulable|type)=="boolean" and (.reason|length)>=8' <<<"$plan" >/dev/null
else
  echo 'Usage: node-cordon.sh plan NODE true|false "reason" or execute PLAN.json' >&2;exit 1
fi
node=$(jq -r .node <<<"$plan")
patch=$(jq -c '[{op:"test",path:"/metadata/uid",value:.uid},{op:"test",path:"/metadata/resourceVersion",value:.resourceVersion},{op:"add",path:"/spec/unschedulable",value:.unschedulable}]' <<<"$plan")
"${k[@]}" patch node "$node" --type=json -p "$patch" --dry-run=server >/dev/null
if [[ "$mode" == execute ]]; then
  "${k[@]}" patch node "$node" --type=json -p "$patch" >/dev/null
  jq '.+{dryRun:false,status:"Accepted",actor:"kubevista-node-operator",note:"Cordon changes scheduling only; it does not drain or evict pods."}' <<<"$plan"
else
  jq '.+{dryRun:true}' <<<"$plan"
fi
