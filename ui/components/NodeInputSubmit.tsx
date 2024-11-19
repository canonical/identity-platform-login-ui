import { getNodeLabel } from "@ory/integrations/ui";
import {
  Button,
  CheckboxInput,
  List,
  Modal,
} from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import React, { FC } from "react";
import { useRouter } from "next/router";

export const NodeInputSubmit: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  const router = useRouter();
  const [saved, setSaved] = React.useState(false);
  const [hasModal, setHasModal] = React.useState(false);

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
  const showTotpLink =
    (node.meta.label as unknown as { hasTotpLink: boolean })?.hasTotpLink ??
    false;
  const showBackLink = node.meta?.label?.text === "Reset password";
  const showBackupCodesIntro =
    (node.meta.label?.context as unknown as { showBackupCodesIntro: boolean })
      ?.showBackupCodesIntro ?? false;
  const hasSavedCodeCheckbox =
    (node.meta.label?.context as unknown as { hasSavedCodeCheckbox: boolean })
      ?.hasSavedCodeCheckbox ?? false;
  const getAppearance = () => {
    const appearance = (
      node.meta.label?.context as unknown as { appearance: string }
    )?.appearance;
    if (appearance !== undefined) {
      return appearance;
    }
    return node.group === "password" ||
      node.group === "code" ||
      node.group === "totp" ||
      node.group === "webauthn" ||
      node.group === "lookup_secret"
      ? "positive"
      : "";
  };
  const hasConfirmBackupCodeModal =
    (
      node.meta.label?.context as unknown as {
        hasConfirmBackupCodeModal: boolean;
      }
    )?.hasConfirmBackupCodeModal ?? false;

  return (
    <>
      {showBackupCodesIntro && (
        <div className="u-sv2">
          Backup codes are on-time passwords you can use to log in when your
          main two-factor authentication method isn{"'"}t available.
        </div>
      )}
      {hasSavedCodeCheckbox && (
        <div className="u-sv3 u-no-print checkbox-save-backup-codes">
          <CheckboxInput
            label="I saved the backup codes"
            onClick={() => setSaved(!saved)}
          />
        </div>
      )}
      {showBackLink && (
        <Button
          tabIndex={3}
          type="button"
          onClick={() => {
            void window.history.back();
          }}
        >
          Back
        </Button>
      )}
      <Button
        appearance={getAppearance()}
        tabIndex={4}
        onClick={async (e) => {
          if (hasConfirmBackupCodeModal) {
            e.preventDefault();
            setHasModal(true);
            return;
          }
          // On click, we set this value, and once set, dispatch the submission!
          await setValue(attributes.value as string).then(() =>
            dispatchSubmit(e),
          );
        }}
        disabled={
          attributes.disabled || disabled || (hasSavedCodeCheckbox && !saved)
        }
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
          tabIndex={5}
          type="button"
          onClick={() => {
            const newQuery = { ...router.query, ["use_backup_code"]: "true" };
            void router.push(
              {
                pathname: window.location.pathname,
                query: newQuery,
              },
              undefined,
              { shallow: true },
            );
          }}
        >
          Use backup code instead
        </Button>
      )}
      {showTotpLink && (
        <Button
          appearance="link"
          tabIndex={5}
          type="button"
          onClick={() => {
            const newQuery = { ...router.query };
            delete newQuery.use_backup_code;
            void router.push(
              {
                pathname: window.location.pathname,
                query: newQuery,
              },
              undefined,
              { shallow: true },
            );
          }}
        >
          Use authentication code instead
        </Button>
      )}
      {hasModal && (
        <Modal
          buttonRow={
            <>
              <Button
                className="u-no-margin--bottom"
                onClick={(e) => {
                  e.preventDefault();
                  setHasModal(false);
                }}
                appearance="base"
              >
                Cancel
              </Button>
              <Button
                appearance="negative"
                className="u-no-margin--bottom"
                onClick={(e) => {
                  e.preventDefault();
                  void setValue(attributes.value as string).then(() =>
                    dispatchSubmit(e),
                  );
                }}
              >
                Deactivate backup codes
              </Button>
            </>
          }
          title="Deactivate backup codes"
        >
          You{"'"}re about to deactivate your backup codes. Please be aware of
          the following:
          <List
            className="deactivate-backup-codes-list"
            items={[
              "All your current backup codes will no longer work.",
              "You won't be able to use backup codes to access your account if you lose access to your primary authentication method.",
              "You'll need to set up new backup codes if you want to use them again in the future.",
            ]}
          />
        </Modal>
      )}
    </>
  );
};
