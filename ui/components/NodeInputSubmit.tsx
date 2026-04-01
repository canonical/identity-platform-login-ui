import { getNodeLabel } from "@ory/integrations/ui";
import { Button, Link } from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import React, { FC } from "react";
import { getProviderImage } from "../util/logos";
import {
  isRegisterEmailSubmit,
  isResendVerificationCode,
  isSignInWithPassword,
} from "../util/constants";
import { ORY_LABEL_CONTINUE_IDENTIFIER_FIRST_LOGIN } from "../util/constants";

function getLoginStartUrl(): string {
  const url = new URL("./login", window.location.href);
  url.searchParams.delete("flow");
  return url.pathname + url.search;
}

export const NodeInputSubmit: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  const label = getNodeLabel(node);
  const isProvider = attributes.name === "provider";
  const provider = attributes.value as string;
  const image = getProviderImage(provider);
  const getAppearance = () => {
    const appearance = (node.meta.label?.context as { appearance: string })
      ?.appearance;
    if (appearance !== undefined) {
      return appearance;
    }
    return node.group === "password" ||
      node.group === "profile" ||
      node.group === "code" ||
      node.group === "identifier_first" ||
      node.group === "totp" ||
      node.group === "webauthn" ||
      node.group === "lookup_secret"
      ? "positive"
      : "";
  };

  const onClick = (
    node.meta.label?.context as {
      onClick: () => void;
    }
  )?.onClick;

  const renderRegistrationCta = () => {
    const isIdentifierFirstSubmit =
      node.group === "identifier_first" &&
      attributes.type === "submit" &&
      node.meta.label?.id === ORY_LABEL_CONTINUE_IDENTIFIER_FIRST_LOGIN;

    if (!isIdentifierFirstSubmit) return null;

    return (
      <p className="registration-cta">
        Don&apos;t have an account? <Link href={`./register`}>Register</Link>
      </p>
    );
  };

  const beforeComponent = (
    node.meta.label?.context as {
      beforeComponent: React.ReactNode;
    }
  )?.beforeComponent;

  const afterComponent = (
    node.meta.label?.context as {
      afterComponent: React.ReactNode;
    }
  )?.afterComponent;

  if (isResendVerificationCode(node)) return null;

  return (
    <>
      {beforeComponent}
      <Button
        appearance={getAppearance()}
        tabIndex={4}
        onClick={async (e) => {
          e.preventDefault();
          e.stopPropagation();

          if (onClick) {
            onClick();
            return;
          }
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
      {isRegisterEmailSubmit(node) && (
        <>
          Already have an account?{" "}
          <Link href={getLoginStartUrl()}>Sign in</Link>
        </>
      )}
      {afterComponent}
      {renderRegistrationCta()}
      {isSignInWithPassword(node) && (
        <Link
          href={`./reset_email?return_to=${encodeURIComponent(getLoginStartUrl())}`}
          tabIndex={3}
        >
          Reset password
        </Link>
      )}
    </>
  );
};
