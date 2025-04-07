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
  isBackupCodeConfirm,
  isBackupCodeConfirmText,
  isBackupCodeCreate,
  isBackupCodeDeactivate,
  isBackupCodeView,
} from "../util/constants";
import { BackupCodeDeletionModal } from "../components/BackupCodeDeletionModal";
import { BackupIntro } from "../components/BackupIntro";
import { BackupCodeSavedCheckbox } from "../components/BackupCodeSavedCheckbox";
import {
  getLoggedInName,
  hasSelfServeReturn,
  formatReturnTo,
} from "../util/selfServeHelpers";

interface Props {
  forceSelfServe: boolean;
}

const SetupBackupCodes: NextPage<Props> = ({ forceSelfServe }: Props) => {
  const [flow, setFlow] = useState<SettingsFlow>();
  const [hasDeletionModal, setHasDeletionModal] = React.useState(false);
  const [hasSavedCodes, setSavedCodes] = React.useState(false);

  // Get ?flow=... from the URL
  const router = useRouter();
  const { return_to: returnTo, flow: flowId } = router.query;

  const isSelfServe = forceSelfServe || hasSelfServeReturn(flow);
  const userName = getLoggedInName(flow);

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
        returnTo: formatReturnTo(returnTo, isSelfServe),
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
          if (methodValues.lookup_secret_confirm && !isSelfServe) {
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
          if (isBackupCodeCreate(node)) {
            node.meta.label.text = "Create backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              beforeComponent: <BackupIntro />,
            };
          }
          if (isBackupCodeCreate(node) && hasDisableCodes) {
            node.meta.label.text = "Create new backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "",
            };
          }
          if (isBackupCodeConfirmText(node)) {
            node.meta.label.text =
              "These are your backup codes. Each backup code can be used once. Store these in a secure place.";
          }
          if (isBackupCodeConfirm(node)) {
            node.meta.label.text = "Create backup codes";
            (node.attributes as UiNodeInputAttributes).disabled =
              !hasSavedCodes;
            node.meta.label.context = {
              ...node.meta.label.context,
              beforeComponent: (
                <BackupCodeSavedCheckbox
                  isChecked={hasSavedCodes}
                  toggleChecked={() => setSavedCodes(!hasSavedCodes)}
                />
              ),
            };
          }
          if (isBackupCodeView(node)) {
            node.meta.label.text = "View backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "",
              beforeComponent: <BackupIntro />,
            };
          }
          if (isBackupCodeDeactivate(node)) {
            node.meta.label.text = "Deactivate backup codes";
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "negative",
              onClick: () => {
                setHasDeletionModal(true);
              },
              afterComponent: (
                <BackupCodeDeletionModal
                  hasModal={hasDeletionModal}
                  onCancel={() => setHasDeletionModal(false)}
                  onConfirm={() =>
                    void handleSubmit({
                      method: "lookup_secret",
                      lookup_secret_disable: true,
                    })
                  }
                />
              ),
            };
            hasDisableCodes = true;
          }
          return node;
        })
        .sort((a, b) => {
          if (isBackupCodeCreate(a) && isBackupCodeDeactivate(b)) {
            return -1;
          }
          if (isBackupCodeDeactivate(a) && isBackupCodeCreate(b)) {
            return 1;
          }
          return 0;
        }),
    },
  } as SettingsFlow;

  return (
    <PageLayout title="Backup codes" isSelfServe={isSelfServe} user={userName}>
      {flow ? <Flow onSubmit={handleSubmit} flow={lookupFlow} /> : <Spinner />}
    </PageLayout>
  );
};

export default SetupBackupCodes;
