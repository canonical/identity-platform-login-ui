import {
  SettingsFlow,
  UpdateSettingsFlowBody,
  UpdateSettingsFlowWithLookupMethod,
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

const SetupBackupCodes: NextPage = () => {
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
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if (data.request_url !== undefined) {
          const pwParam = pwChanged
            ? `&pw_changed=${pwChanged.toString()}`
            : "";
          window.location.href = `./setup_backup_codes?flow=${data.id}${pwParam}`;
          return;
        }
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch(async (err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
          setFlow(undefined);
          window.location.href = "./login";
          return;
        }

        return Promise.reject(err);
      });
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateSettingsFlowBody) => {
      console.log(values);
      const methodValues = values as UpdateSettingsFlowWithLookupMethod;
      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            csrf_token: (flow?.ui?.nodes[0].attributes as UiNodeInputAttributes)
              .value as string,
            method: "lookup_secret",
            lookup_secret_reveal: methodValues.lookup_secret_reveal
              ? true
              : undefined,
            lookup_secret_confirm: methodValues.lookup_secret_confirm
              ? true
              : undefined,
            lookup_secret_regenerate: methodValues.lookup_secret_regenerate
              ? true
              : undefined,
            lookup_secret_disable: methodValues.lookup_secret_disable
              ? true
              : undefined,
          },
        })
        .then(({ data }) => {
          if (flow?.state === "success") {
            window.location.href = "./setup_complete";
          }
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
          window.location.href = "./error";
        })
        .catch(handleFlowError("settings", setFlow));
    },
    [flow, router],
  );

  const lookupFlow = {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes.filter(({ group }) => {
        return group === "lookup_secret";
      }),
    },
  } as SettingsFlow;

  return (
    <PageLayout title="Set up backup codes">
      {pwChanged === "success" && (
        <Notification severity="positive">
          Password was changed successfully
        </Notification>
      )}
      {flow ? <Flow onSubmit={handleSubmit} flow={lookupFlow} /> : <Spinner />}
    </PageLayout>
  );
};

export default SetupBackupCodes;
