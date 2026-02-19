export function setFlowIDQueryParam(flowId: string) {
  const url = new URL(window.location.href);
  url.searchParams.set("flow", flowId);
  window.history.replaceState(null, "", url);
}
