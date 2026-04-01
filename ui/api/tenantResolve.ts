type ResolveResult = {
  redirect_to?: string;
};

/**
 * Calls the backend resolve endpoint to determine tenant routing.
 * Returns a redirect URL if the user needs to be sent somewhere (single-tenant
 * callback or multi-tenant selection page), or undefined if the caller should
 * proceed with the original flow response.
 */
export const resolveTenant = (
  flowId: string,
  loginChallenge: string,
): Promise<string | undefined> => {
  return fetch("/api/v0/tenants/resolve", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ flow: flowId, login_challenge: loginChallenge }),
  })
    .then((r) => {
      if (!r.ok) {
        throw new Error(`Tenant resolve returned ${r.status}`);
      }
      return r.json() as Promise<ResolveResult>;
    })
    .then((result) => result.redirect_to);
};
