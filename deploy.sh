#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="${ROOT_DIR}/k8s"

KUBECTL_BIN="${KUBECTL_BIN:-kubectl}"
NAMESPACE="${NAMESPACE:-default}"

die() {
  echo "error: $*" >&2
  exit 1
}

need_file() {
  [[ -f "$1" ]] || die "missing file: $1"
}

need_file "${K8S_DIR}/configmap.yaml"
need_file "${K8S_DIR}/pod.yaml"
need_file "${K8S_DIR}/deployment.yaml"
need_file "${K8S_DIR}/service.yaml"
need_file "${K8S_DIR}/daemonset-log-agent.yaml"
need_file "${K8S_DIR}/cronjob-archive-logs.yaml"

echo "Using kubectl: ${KUBECTL_BIN}"
echo "Namespace: ${NAMESPACE}"

if ! "${KUBECTL_BIN}" version --client >/dev/null 2>&1; then
  die "kubectl is not available"
fi

echo "Checking cluster connectivity..."
if ! "${KUBECTL_BIN}" -n "${NAMESPACE}" get ns "${NAMESPACE}" >/dev/null 2>&1; then
  die "cannot access cluster or namespace '${NAMESPACE}'"
fi

echo "Applying manifests..."
${KUBECTL_BIN} -n "${NAMESPACE}" apply -f "${K8S_DIR}/configmap.yaml"
${KUBECTL_BIN} -n "${NAMESPACE}" apply -f "${K8S_DIR}/service.yaml"
${KUBECTL_BIN} -n "${NAMESPACE}" apply -f "${K8S_DIR}/deployment.yaml"
${KUBECTL_BIN} -n "${NAMESPACE}" apply -f "${K8S_DIR}/daemonset-log-agent.yaml"
${KUBECTL_BIN} -n "${NAMESPACE}" apply -f "${K8S_DIR}/cronjob-archive-logs.yaml"

${KUBECTL_BIN} -n "${NAMESPACE}" apply -f "${K8S_DIR}/pod.yaml"

echo "Waiting for resources to become ready..."
${KUBECTL_BIN} -n "${NAMESPACE}" wait --for=condition=Ready pod/app-pod --timeout=120s || true
${KUBECTL_BIN} -n "${NAMESPACE}" rollout status deployment/app --timeout=180s
${KUBECTL_BIN} -n "${NAMESPACE}" rollout status daemonset/log-agent --timeout=180s

echo
echo "Deployed."
echo "- Pods:"
${KUBECTL_BIN} -n "${NAMESPACE}" get pods -l app=app
echo "- Service:"
${KUBECTL_BIN} -n "${NAMESPACE}" get svc app
echo "- DaemonSet:"
${KUBECTL_BIN} -n "${NAMESPACE}" get ds log-agent
echo "- CronJob:"
${KUBECTL_BIN} -n "${NAMESPACE}" get cronjob app-logs-archive

cat <<'EOF'

Next steps:
- Access API from your host:
  kubectl port-forward svc/app 8280:8280
  curl http://127.0.0.1:8280/status

EOF

