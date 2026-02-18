import { Configuration, FrontendApi, UpdateLoginFlowBody, LoginFlow } from "@ory/client";

export const kratos = new FrontendApi(
  new Configuration({
    // WIP needs to be configurable
    basePath: "..",
    baseOptions: {
      withCredentials: true,
    },
  })
);

type IdentifierFirstResponse = { redirect_to: string } | LoginFlow;

export async function loginIdentifierFirst(
  flowId: string,
  values: UpdateLoginFlowBody,
  method: string,
  flow?: { id?: string; return_to?: string }
) {
  const res = await fetch(
    `/self-service/login/id-first?flow=${encodeURIComponent(flowId)}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        ...values,
        method,
        flow: String(flow?.id),
      }),
    },
  );

  if (!res.ok) {
    throw new Error(await res.text());
  }

  return (await res.json()) as IdentifierFirstResponse;
}
