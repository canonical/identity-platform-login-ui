import { Configuration, FrontendApi, UpdateLoginFlowBody, LoginFlow } from "@ory/client";

export const kratos = new FrontendApi(
  new Configuration({
    // Use relative path so that this works when served in a subpath
    basePath: "../api/kratos",
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
    `/api/kratos/self-service/login/id-first?flow=${encodeURIComponent(flowId)}`,
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
