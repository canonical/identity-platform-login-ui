import {
  Configuration,
  FrontendApi,
  UpdateLoginFlowBody,
  LoginFlow,
} from "@ory/client";

const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "";

export const kratos = new FrontendApi(
  new Configuration({
    basePath: apiBase || "..",
    baseOptions: {
      withCredentials: true,
    },
  }),
);

type IdentifierFirstResponse = { redirect_to: string } | LoginFlow;

export async function loginIdentifierFirst(
  flowId: string,
  values: UpdateLoginFlowBody,
  method: string,
  flow?: { id?: string; return_to?: string },
  loginChallenge?: string,
) {
  const params = new URLSearchParams({ flow: flowId });
  if (loginChallenge) {
    params.set("login_challenge", loginChallenge);
  }
  const endpoint = `${apiBase}/self-service/login/id-first?${params.toString()}`;
  const res = await fetch(endpoint, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      ...values,
      method,
      flow: String(flow?.id),
    }),
  });

  if (!res.ok) {
    throw new Error(await res.text());
  }

  return (await res.json()) as IdentifierFirstResponse;
}
