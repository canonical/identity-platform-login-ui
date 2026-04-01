import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useCallback, useEffect, useState } from "react";
import React from "react";
import { Button, Notification, Spinner } from "@canonical/react-components";
import {
  Tenant,
  fetchTenantsByFlow,
  fetchTenantsBySession,
} from "../api/tenants";
import PageLayout from "../components/PageLayout";

const SelectTenant: NextPage = () => {
  const router = useRouter();
  const { flow: flowId } = router.query;
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submitTenantSelection = useCallback(
    (tenantId: string) => {
      const { flow, login_challenge } = router.query;

      if (!login_challenge || typeof login_challenge !== "string") {
        return;
      }

      const body: Record<string, string> = {
        login_challenge,
        tenant_id: tenantId,
      };
      if (flow && typeof flow === "string") {
        body.flow = flow;
      }

      void fetch("/api/v0/auth/tenant", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      })
        .then((r) => {
          if (!r.ok) {
            throw new Error(`failed to store tenant selection: ${r.status}`);
          }
          return r.json() as Promise<{ redirect_to: string }>;
        })
        .then((r) => {
          window.location.href = r.redirect_to;
        })
        .catch((err: Error) => {
          console.error(err);
          setError("Failed to select tenant. Please try again.");
        });
    },
    [router.query],
  );

  useEffect(() => {
    if (!router.isReady) return;
    setLoading(true);
    const loader =
      typeof flowId === "string"
        ? fetchTenantsByFlow(flowId)
        : fetchTenantsBySession();
    loader
      .then((result) => {
        setTenants(result);
        if (result.length === 0) {
          submitTenantSelection("");
        }
      })
      .catch((err) => {
        console.error(err);
        setError(
          "Failed to fetch tenants. Ensure the backend is running and the user exists.",
        );
      })
      .finally(() => setLoading(false));
  }, [router.isReady, flowId, submitTenantSelection]);

  if (!router.isReady) return null;

  return (
    <PageLayout title="Select a tenant">
      {loading && (
        <div className="u-align--center">
          <Spinner text="Loading tenants…" />
        </div>
      )}
      {error && (
        <Notification severity="negative" inline>
          {error}
        </Notification>
      )}
      {!loading && !error && tenants.length === 0 && (
        <Spinner text="Completing login…" />
      )}
      {!loading && tenants.length > 0 && (
        <ul className="p-list">
          {tenants.map((tenant) => (
            <li key={tenant.id} className="p-list__item">
              <Button
                className="u-no-margin--bottom p-select-tenant__button"
                onClick={() => submitTenantSelection(tenant.id)}
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
