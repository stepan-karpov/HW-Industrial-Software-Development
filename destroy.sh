#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="${ROOT_DIR}/k8s"

KUBECTL_BIN="${KUBECTL_BIN:-kubectl}"
NAMESPACE="${NAMESPACE:-default}"

# kind cluster name (если используете kind)
KIND_BIN="${KIND_BIN:-kind}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-hw-app}"

die() {
  echo "error: $*" >&2
  exit 1
}

has() {
  command -v "$1" >/dev/null 2>&1
}

echo "Namespace: ${NAMESPACE}"

if has "${KUBECTL_BIN}"; then
  if "${KUBECTL_BIN}" version --client >/dev/null 2>&1; then
    echo "Deleting Kubernetes resources (ignore not found)..."

    # Сначала останавливаем планировщик CronJob'ов и удаляем возможные Job'ы, чтобы не мешали удалению.
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete cronjob app-logs-archive --ignore-not-found || true
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete job -l app=app-logs-archive --ignore-not-found || true
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete job app-logs-archive-manual --ignore-not-found || true

    # Удаляем объекты из манифестов (порядок: consumer -> provider)
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete -f "${K8S_DIR}/pod.yaml" --ignore-not-found || true
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete -f "${K8S_DIR}/deployment.yaml" --ignore-not-found || true
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete -f "${K8S_DIR}/daemonset-log-agent.yaml" --ignore-not-found || true
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete -f "${K8S_DIR}/service.yaml" --ignore-not-found || true
    "${KUBECTL_BIN}" -n "${NAMESPACE}" delete -f "${K8S_DIR}/configmap.yaml" --ignore-not-found || true

    echo "Kubernetes resources deleted."
  else
    echo "kubectl found but not functional; skipping Kubernetes cleanup." >&2
  fi
else
  echo "kubectl not found; skipping Kubernetes cleanup." >&2
fi

if has "${KIND_BIN}"; then
  echo "Deleting kind cluster '${KIND_CLUSTER_NAME}' (stops/removes nodes)..."
  "${KIND_BIN}" delete cluster --name "${KIND_CLUSTER_NAME}" || true
else
  echo "kind not found; skipping node shutdown (kind delete cluster)." >&2
fi

echo "Done."

