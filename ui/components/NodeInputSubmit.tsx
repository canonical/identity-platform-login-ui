import { getNodeLabel } from "@ory/integrations/ui";
import { Button, Link } from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import React, { Component, FC } from "react";
import { getProviderImage } from "../util/logos";
import { ORY_LABEL_CONTINUE_IDENTIFIER_FIRST_LOGIN } from "../util/constants";

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
        Don&apos;t have an account?{" "}
        <Link href="/ui/register_email">Register</Link>
      </p>
    );
  };

  const beforeComponent = (
    node.meta.label?.context as {
      beforeComponent: Component;
    }
  )?.beforeComponent;

  const afterComponent = (
    node.meta.label?.context as {
      afterComponent: Component;
    }
  )?.afterComponent;

  return (
    <>
      {beforeComponent}
      <Button
        appearance={getAppearance()}
        tabIndex={4}
        onClick={async (e) => {
          if (onClick) {
            e.preventDefault();
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
      {afterComponent}
      {renderRegistrationCta()}
    </>
  );
};
