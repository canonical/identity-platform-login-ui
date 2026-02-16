export function setFlowIDQueryParam(flowId: string) {
  window.history.replaceState(null, "", `?flow=${flowId}`);
}
