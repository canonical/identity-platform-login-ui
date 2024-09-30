import {
  SettingsFlow,
  UpdateSettingsFlowBody,
  UpdateSettingsFlowWithTotpMethod,
} from "@ory/client";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import { AxiosError } from "axios";
import { Notification, Spinner } from "@canonical/react-components";
import { UiNodeInputAttributes } from "@ory/client/api";

const SetupSecure: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const {
    return_to: returnTo,
    flow: flowId,
    pw_changed: pwChanged,
  } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    // If ?flow=... was in the URL, we fetch it
    if (flowId) {
      kratos
        .getSettingsFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("settings", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserSettingsFlow({
        returnTo: returnTo
          ? returnTo.toString()
          : window.location.pathname.replace("setup_secure", "setup_complete"),
      })
      .then(({ data }) => {
        const pwParam = pwChanged ? `&pw_changed=${pwChanged.toString()}` : "";
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./reset_password?flow=${data.id}${pwParam}`,
          );
        }
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch(async (err: AxiosError<string>) => {
        return Promise.reject(err);
      });
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateSettingsFlowBody) => {
      const methodValues = values as UpdateSettingsFlowWithTotpMethod;
      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            csrf_token: (flow?.ui?.nodes[0].attributes as UiNodeInputAttributes)
              .value as string,
            method: "totp",
            totp_code: methodValues.totp_code,
            totp_unlink: methodValues.totp_unlink ? true : undefined,
          },
        })
        .then(({ data }) => {
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }
          if (data?.return_to) {
            window.location.href = data?.return_to;
            return;
          }
          window.location.href = "./error";
        })
        .catch(handleFlowError("settings", setFlow));
    },
    [flow, router],
  );

  const totpFlow = {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes.filter(({ group }) => {
        return group === "totp";
      }),
    },
  } as SettingsFlow;

  return (
    <PageLayout title="Secure your account">
      {pwChanged === "success" && (
        <Notification severity="positive">
          Password was changed successfully
        </Notification>
      )}
      {flow ? <Flow onSubmit={handleSubmit} flow={totpFlow} /> : <Spinner />}
    </PageLayout>
  );
};

export default SetupSecure;
