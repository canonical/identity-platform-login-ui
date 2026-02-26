import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import React from "react";
import { Button, Spinner } from "@canonical/react-components";
import { Tenant, fetchTenants } from "../api/tenants";
import PageLayout from "../components/PageLayout";

const SelectTenant: NextPage = () => {
  const router = useRouter();
  const { user_id } = router.query;
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!user_id || typeof user_id !== "string") return;

    setLoading(true);
    fetchTenants(user_id)
      .then(setTenants)
      .catch((err) => {
        console.error(err);
        setError(
          "Failed to fetch tenants. Ensure the backend is running and the user exists.",
        );
      })
      .finally(() => setLoading(false));
  }, [user_id]);

  const handleSelect = (tenantId: string) => {
    const { redirect_to, flow } = router.query;
    if (redirect_to && typeof redirect_to === "string") {
      localStorage.setItem("selected_tenant_id", tenantId);
      window.location.href = redirect_to;
    } else {
      void router.push({
        pathname: "/login",
        query: { tenant_id: tenantId, ...(flow ? { flow } : {}) },
      });
    }
  };

  if (!router.isReady) return null;

  if (!user_id) {
    return (
      <PageLayout title="Select Tenant">
        <p>Invalid request: Missing User ID</p>
      </PageLayout>
    );
  }

  return (
    <PageLayout title="Select a tenant">
      {loading && (
        <div className="u-align--center">
          <Spinner text="Loading tenants&hellip;" />
        </div>
      )}
      {error && <p className="p-notification--negative">{error}</p>}
      {!loading && !error && tenants.length === 0 && (
        <p>No tenants found for this user.</p>
      )}
      {!loading && tenants.length > 0 && (
        <ul className="p-list">
          {tenants.map((tenant) => (
            <li key={tenant.id} className="p-list__item">
              <Button
                className="u-no-margin--bottom"
                style={{ width: "100%", textAlign: "left" }}
                onClick={() => handleSelect(tenant.id)}
              >
                {tenant.name}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </PageLayout>
  );
};

export default SelectTenant;
