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

export const fetchTenantsByFlow = (flowId: string): Promise<Tenant[]> =>
  fetch(`/api/v0/tenants?flow=${encodeURIComponent(flowId)}`).then(parseTenants);

export const fetchTenantsBySession = (): Promise<Tenant[]> =>
  fetch("/api/v0/tenants").then(parseTenants);
