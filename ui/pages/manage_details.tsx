import type { NextPage } from "next";
import React, { useEffect, useState } from "react";
import PageLayout from "../components/PageLayout";
import { useRouter } from "next/router";
import { Input } from "@canonical/react-components";
import { kratos } from "../api/kratos";
import { handleFlowError } from "../util/handleFlowError";
import { SettingsFlow } from "@ory/client";
import { AxiosError } from "axios";
import {
  getFullName,
  getLoggedInName,
  IdentityTraits,
} from "../util/selfServeHelpers";

const ManageDetails: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const { return_to: returnTo, flow: flowId } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    // If ?flow=... was in the URL, we fetch it
    if (flowId) {
      kratos
        .getSettingsFlow({ id: String(flowId) })
        .then((res) => {
          setFlow(res.data);
        })
        .catch(handleFlowError("settings", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserSettingsFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./manage_details?flow=${data.id}`,
          );
          router.query.flow = data.id;
        }
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch(async (err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
          window.location.href = `./login?return_to=${window.location.pathname}`;
          return;
        }

        return Promise.reject(err);
      });
  }, [flowId, router, router.isReady, returnTo, flow]);

  const userName = getLoggedInName(flow);
  const fullName = getFullName(flow);
  const traits = flow?.identity?.traits as IdentityTraits;
  const email = traits?.email ?? "";

  return (
    <PageLayout title="Details" isSelfServe={true} user={userName}>
      <Input type="text" value={email} disabled label="Email address" />
      {fullName && (
        <Input type="text" value={fullName} disabled label="Full name" />
      )}
    </PageLayout>
  );
};

export default ManageDetails;
