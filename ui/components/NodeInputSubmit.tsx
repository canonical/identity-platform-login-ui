import { getNodeLabel } from "@ory/integrations/ui";
import { Button, Link } from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import React, { FC, useContext } from "react";
import { getProviderImage } from "../util/logos";
import {
  isRegisterEmailSubmit,
  isResendVerificationCode,
  isSignInWithPassword,
  isVerificationCodeInput,
} from "../util/constants";
import { FlowContext } from "../context/FlowContext";
import { ORY_LABEL_CONTINUE_IDENTIFIER_FIRST_LOGIN } from "../util/constants";

// The Hydra login and the return URL the login page was opened with, so a user who registers
// instead of logging in still finishes that login.
function getRegisterUrl(): string {
  const current = new URLSearchParams(window.location.search);
  const params = new URLSearchParams();
  for (const name of ["login_challenge", "return_to"]) {
    const value = current.get(name);
    if (value) {
      params.set(name, value);
    }
  }
  const query = params.toString();
  return query ? `./register?${query}` : "./register";
}

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
  const flow = useContext(FlowContext);
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
        Don&apos;t have an account?{" "}
        <Link href={getRegisterUrl()}>Register</Link>
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

  // The verification code input renders its own inline "Resend code" link
  // (see NodeInputText), so only hide the button when that input is present.
  if (
    isResendVerificationCode(node) &&
    flow.ui?.nodes.some(isVerificationCodeInput)
  ) {
    return null;
  }

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
