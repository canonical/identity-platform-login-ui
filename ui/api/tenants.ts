export type Tenant = {
  id: string;
  name: string;
};

const parseTenants = (r: Response): Promise<Tenant[]> => {
  if (!r.ok) {
    throw new Error(`Tenants API returned ${r.status}`);
  }
  return (r.json() as Promise<{ tenants: Tenant[] }>).then(
    (body) => body.tenants,
  );
};

const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "";

export const fetchTenantsByFlow = (flowId: string): Promise<Tenant[]> =>
  fetch(`${apiBase}/api/v0/tenants?flow=${encodeURIComponent(flowId)}`).then(
    parseTenants,
  );

export const fetchTenantsBySession = (): Promise<Tenant[]> =>
  fetch(`${apiBase}/api/v0/tenants`).then(parseTenants);
