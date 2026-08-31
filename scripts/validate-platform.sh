#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
validation_dir="${repository_root}/work/platform-validation"
mkdir -p "${validation_dir}"

export HELM_CONFIG_HOME="${validation_dir}/helm/config"
export HELM_CACHE_HOME="${validation_dir}/helm/cache"
export HELM_DATA_HOME="${validation_dir}/helm/data"
mkdir -p "${HELM_CONFIG_HOME}" "${HELM_CACHE_HOME}" "${HELM_DATA_HOME}"

helm repo add eks https://aws.github.io/eks-charts --force-update >/dev/null
helm repo add external-secrets https://charts.external-secrets.io --force-update >/dev/null
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/ --force-update >/dev/null
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update >/dev/null
helm repo add grafana-community https://grafana-community.github.io/helm-charts --force-update >/dev/null
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts --force-update >/dev/null
helm repo update >/dev/null

extract_values() {
  local application_file="$1"
  local values_file="$2"
  awk 'BEGIN { found=0 } /values: \|/ { found=1; next } found && /^  destination:/ { exit } found { sub(/^        /, ""); print }' \
    "${application_file}" > "${values_file}"
}

render() {
  local application="$1"
  local release="$2"
  local chart="$3"
  local version="$4"
  local namespace="$5"
  local values_file="${validation_dir}/${application}.values.yaml"
  extract_values "${repository_root}/platform/argocd/applications/${application}.yaml" "${values_file}"
  helm template "${release}" "${chart}" --version "${version}" --namespace "${namespace}" -f "${values_file}" >/dev/null
}

helm lint "${repository_root}/platform/apps/dashboard"
helm lint "${repository_root}/platform/apps/platform-config"
render 10-aws-load-balancer-controller aws-load-balancer-controller eks/aws-load-balancer-controller 3.5.0 kube-system
render 11-external-secrets external-secrets external-secrets/external-secrets 2.10.0 external-secrets
render 12-metrics-server metrics-server metrics-server/metrics-server 3.14.0 kube-system
render 30-loki loki grafana-community/loki 18.11.7 observability
render 31-tempo tempo grafana-community/tempo 2.3.0 observability
render 32-monitoring monitoring prometheus-community/kube-prometheus-stack 88.6.1 observability
render 33-otel-gateway otel-gateway open-telemetry/opentelemetry-collector 0.172.0 observability
render 34-otel-agent otel-agent open-telemetry/opentelemetry-collector 0.172.0 observability
