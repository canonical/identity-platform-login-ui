export type Tenant = {
  id: string;
  name: string;
  enabled: boolean;
};

export const fetchTenants = (userId: string): Promise<Tenant[]> => {
  return fetch(`/api/v0/users/${userId}/tenants`, {
    headers: { Authorization: "Bearer a" },
  }).then((r) => {
    if (!r.ok) {
      throw new Error(`Tenants API returned ${r.status}`);
    }
    return r.json() as Promise<Tenant[]>;
  });
};
