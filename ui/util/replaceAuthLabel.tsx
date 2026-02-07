import React from "react";
import { LoginFlow } from "@ory/client";
import { UseBackupCodeButton } from "../components/UseBackupCodeButton";
import { UseOtherButton } from "../components/UseOtherButton";
import {
  isSignInWithHardwareKey,
  isUseAuthenticator,
  isUseBackupCode,
} from "./constants";

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
  const hasConfiguredWebauth = flow.ui.nodes.some(
    (node) => node.group === "webauthn",
  );

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
                    <UseOtherButton method="authentication code" />
                  ) : hasConfiguredWebauth ? (
                    <UseOtherButton method="security key" />
                  ) : undefined,
                },
              },
            },
          };
        }
        if (isSignInWithHardwareKey(node)) {
          return {
            ...node,
            meta: {
              ...node.meta,
              label: {
                ...node.meta.label,
                context: {
                  ...node.meta.label.context,
                  additional: hasConfiguredBackupCodes ? (
                    <UseBackupCodeButton />
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
