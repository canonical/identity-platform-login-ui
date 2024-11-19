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
import { Spinner } from "@canonical/react-components";
import { UiNodeInputAttributes } from "@ory/client/api";
import {
  ORY_LABEL_BACKUP_CODE_CONFIRM,
  ORY_LABEL_BACKUP_CODE_CONFIRM_TEXT,
  ORY_LABEL_BACKUP_CODE_CREATE,
  ORY_LABEL_BACKUP_CODE_DEACTIVATE,
  ORY_LABEL_BACKUP_CODE_VIEW,
} from "../util/constants";

const SetupBackupCodes: NextPage = () => {
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
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./setup_backup_codes?flow=${data.id}`,
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

  const handleSubmit = useCallback(
    (values: UpdateSettingsFlowBody) => {
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
          if (methodValues.lookup_secret_confirm) {
            const flowParam =
              data?.return_to && !data.return_to.endsWith("/setup_complete")
                ? `?flow=${data.id}`
                : "";
            window.location.href = `./setup_complete${flowParam}`;
          } else {
            setFlow(data); // Reset the flow to trigger refresh
          }
        })
        .catch(handleFlowError("settings", setFlow));
    },
    [flow, router],
  );

  let hasDisableCodes = false;
  const lookupFlow = {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes
        .filter(({ group }) => {
          return group === "lookup_secret";
        })
        .map((node) => {
          if (node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CREATE) {
            node.meta.label.text = "Create backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              showBackupCodesIntro: true,
            };
          }
          if (
            node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CREATE &&
            hasDisableCodes
          ) {
            node.meta.label.text = "Create new backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "",
              showBackupCodesIntro: false,
            };
          }
          if (node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CONFIRM_TEXT) {
            node.meta.label.text =
              "These are your back up codes. Each backup code can be used once. Store these in a secure place.";
          }
          if (node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CONFIRM) {
            node.meta.label.text = "Create backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              hasSavedCodeCheckbox: true,
            };
          }
          if (node.meta.label?.id === ORY_LABEL_BACKUP_CODE_VIEW) {
            node.meta.label.text = "View backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "",
              showBackupCodesIntro: true,
            };
          }
          if (node.meta.label?.id === ORY_LABEL_BACKUP_CODE_DEACTIVATE) {
            node.meta.label.text = "Deactivate backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "negative",
              hasConfirmBackupCodeModal: true,
            };
            hasDisableCodes = true;
          }
          return node;
        })
        .sort((a, b) => {
          if (
            a.meta.label?.id === ORY_LABEL_BACKUP_CODE_CREATE &&
            b.meta.label?.id === ORY_LABEL_BACKUP_CODE_DEACTIVATE
          ) {
            return -1;
          }
          if (
            a.meta.label?.id === ORY_LABEL_BACKUP_CODE_DEACTIVATE &&
            b.meta.label?.id === ORY_LABEL_BACKUP_CODE_CREATE
          ) {
            return 1;
          }
          return 0;
        }),
    },
  } as SettingsFlow;

  return (
    <PageLayout title="Backup codes">
      {flow ? <Flow onSubmit={handleSubmit} flow={lookupFlow} /> : <Spinner />}
    </PageLayout>
  );
};

export default SetupBackupCodes;
