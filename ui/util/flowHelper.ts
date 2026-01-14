import {NextRouter} from "next/router";

export function setFlowIDQueryParam(router: NextRouter, flowId: string) {
  void router.push(
    {
      query: {...router.query, flow: flowId},
    },
    undefined,
    {shallow: true},
  );
}
