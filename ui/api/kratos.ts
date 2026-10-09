import {
  Configuration,
  FrontendApi,
  UpdateLoginFlowBody,
  LoginFlow,
} from "@ory/client";

export const kratos = new FrontendApi(
  new Configuration({
    // WIP needs to be configurable
    basePath: "..",
    baseOptions: {
      withCredentials: true,
    },
  }),
);

type IdentifierFirstResponse = { redirect_to: string } | LoginFlow;

// Raised on a non-OK identifier-first answer; keeps the status and the parsed
// JSON body (if any) so callers can react to a specific error id.
export class IdentifierFirstError extends Error {
  status: number;
  data?: unknown;

  constructor(status: number, text: string) {
    super(text);
    this.status = status;
    try {
      this.data = JSON.parse(text);
    } catch {
      this.data = undefined;
    }
  }
}

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
  const res = await fetch(`/self-service/login/id-first?${params.toString()}`, {
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
    throw new IdentifierFirstError(res.status, await res.text());
  }

  return (await res.json()) as IdentifierFirstResponse;
}
