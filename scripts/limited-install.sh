#!/usr/bin/env bash
# Limited-privilege DataRobot install: admin installs datarobot-infra, then a
# namespace-scoped ServiceAccount installs datarobot-prime with no cluster
# privileges via the filter-resources post-renderer.
#
# Required env vars:
#   PRIME_CHART  - path to the datarobot-prime .tgz
#   INFRA_CHART  - path to the generated datarobot-infra .tgz (see:
#                  helm-datarobot infra-chart)
#
# Optional env vars (defaults shown):
#   NAMESPACE         datarobot
#   RELEASE_NAME      datarobot
#   PIPELINE_SA       pipeline
#   KUBECONFIG_ADMIN  ${KUBECONFIG:-${HOME}/.kube/config}
#   KUBECONFIG_LIMITED ${HOME}/.kube/pipeline.yaml
#   VALUES_FILE       (none) - extra -f values file passed to both installs
#   POST_RENDERER     helm-datarobot
#   WAIT_TIMEOUT      300s
set -euo pipefail

PRIME_CHART="${PRIME_CHART:?Set PRIME_CHART to the datarobot-prime .tgz}"
INFRA_CHART="${INFRA_CHART:?Set INFRA_CHART to the generated datarobot-infra .tgz}"
NAMESPACE="${NAMESPACE:-datarobot}"
RELEASE_NAME="${RELEASE_NAME:-datarobot}"
PIPELINE_SA="${PIPELINE_SA:-pipeline}"
KUBECONFIG_ADMIN="${KUBECONFIG_ADMIN:-${KUBECONFIG:-${HOME}/.kube/config}}"
KUBECONFIG_LIMITED="${KUBECONFIG_LIMITED:-${HOME}/.kube/pipeline.yaml}"
VALUES_FILE="${VALUES_FILE:-}"
POST_RENDERER="${POST_RENDERER:-helm-datarobot}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-300s}"

EXTRA=()
[[ -n "${VALUES_FILE}" ]] && EXTRA+=(-f "${VALUES_FILE}")

echo "==> [admin] install ${INFRA_CHART} into ${NAMESPACE}"
KUBECONFIG="${KUBECONFIG_ADMIN}" helm upgrade --install datarobot-infra "${INFRA_CHART}" \
  -n "${NAMESPACE}" --create-namespace "${EXTRA[@]+"${EXTRA[@]}"}"

echo "==> [admin] wait for CRDs Established (required for Envoy Gateway)"
mapfile -t CRDS < <(KUBECONFIG="${KUBECONFIG_ADMIN}" helm get manifest datarobot-infra -n "${NAMESPACE}" \
  | awk '/^kind: CustomResourceDefinition$/{c=1} c&&/^  name:/{print $2; c=0}')
for crd in "${CRDS[@]}"; do
  echo "    waiting on crd/${crd}"
  KUBECONFIG="${KUBECONFIG_ADMIN}" kubectl wait --for=condition=Established "crd/${crd}" --timeout="${WAIT_TIMEOUT}"
done

echo "==> [pipeline] install ${RELEASE_NAME} as limited SA ${PIPELINE_SA} (post-renderer + --skip-crds)"
KUBECONFIG="${KUBECONFIG_LIMITED}" helm upgrade --install "${RELEASE_NAME}" "${PRIME_CHART}" \
  -n "${NAMESPACE}" \
  --skip-crds \
  --post-renderer "${POST_RENDERER}" \
  --post-renderer-args filter-resources \
  --post-renderer-args --keep=app \
  "${EXTRA[@]+"${EXTRA[@]}"}"

echo "==> done. Verifying app manifest is namespace-scoped only:"
KUBECONFIG="${KUBECONFIG_LIMITED}" helm get manifest "${RELEASE_NAME}" -n "${NAMESPACE}" \
  | grep '^kind:' | sort | uniq -c | sort -rn
