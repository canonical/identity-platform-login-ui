import React from "react";
import { LoginFlow } from "@ory/client";
import { UseBackupCodeButton } from "../components/UseBackupCodeButton";
import { UseTotpButton } from "../components/UseTotpButton";
import { isUseAuthenticator, isUseBackupCode } from "./constants";

export const replaceAuthLabel = (
  flow: LoginFlow | undefined,
): LoginFlow | undefined => {
  if (!flow) {
    return flow;
  }

  const hasConfiguredBackupCodes = flow.ui.nodes.some(
    (node) => node.group === "lookup_secret",
  );

  const hasConfiguredTotp = flow.ui.nodes.some((node) => node.group === "totp");

  return {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes.map((node) => {
        if (isUseAuthenticator(node)) {
          return {
            ...node,
            meta: {
              ...node.meta,
              label: {
                ...node.meta.label,
                text: "Sign in",
                context: {
                  ...node.meta.label.context,
                  afterComponent: hasConfiguredBackupCodes ? (
                    <UseBackupCodeButton />
                  ) : undefined,
                },
              },
            },
          };
        }
        if (isUseBackupCode(node)) {
          return {
            ...node,
            meta: {
              ...node.meta,
              label: {
                ...node.meta.label,
                text: "Sign in",
                context: {
                  ...node.meta.label.context,
                  afterComponent: hasConfiguredTotp ? (
                    <UseTotpButton />
                  ) : undefined,
                },
              },
            },
          };
        }
        return node;
      }),
    },
  };
};
