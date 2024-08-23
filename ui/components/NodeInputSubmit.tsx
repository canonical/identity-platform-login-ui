import { getNodeLabel } from "@ory/integrations/ui";
import { Button } from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import React, { FC } from "react";

export const NodeInputSubmit: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  const getProviderImage = (value: string) => {
    if (value.toLowerCase().startsWith("auth0")) {
      return "logos/Auth0.svg";
    }
    if (value.toLowerCase().startsWith("github")) {
      return "logos/Github.svg";
    }
    if (value.toLowerCase().startsWith("google")) {
      return "logos/Google.svg";
    }
    if (value.toLowerCase().startsWith("microsoft")) {
      return "logos/Microsoft.svg";
    }
    if (value.toLowerCase().startsWith("ping")) {
      return "logos/Ping.svg";
    }
    return "logos/Fallback.svg";
  };

  const label = getNodeLabel(node);
  const isProvider = attributes.name === "provider";
  const provider = attributes.value as string;
  const image = getProviderImage(provider);
  const showBackupLink =
    (node.meta.label as unknown as { hasBackupLink: boolean })?.hasBackupLink ??
    false;

  return (
    <>
      <Button
        appearance={
          node.group === "password" ||
          node.group === "code" ||
          node.group === "totp" ||
          node.group === "webauthn" ||
          node.group === "lookup_secret"
            ? "positive"
            : ""
        }
        onClick={async (e) => {
          // On click, we set this value, and once set, dispatch the submission!
          await setValue(attributes.value as string).then(() =>
            dispatchSubmit(e),
          );
        }}
        disabled={attributes.disabled || disabled}
        className={
          node.group === "oidc" ? "oidc-login-button u-no-print" : "u-no-print"
        }
      >
        {isProvider ? (
          <>
            <img
              src={image}
              alt={`${provider} logo`}
              style={{ marginRight: "0.5rem" }}
            />
            <span>Sign in with {label}</span>
          </>
        ) : (
          label
        )}
      </Button>
      {showBackupLink && (
        <Button
          appearance="link"
          type="button"
          onClick={() =>
            (window.location.href = window.location.href + "&use_backup_code=1")
          }
        >
          Use backup code instead
        </Button>
      )}
    </>
  );
};
