import { getNodeLabel } from "@ory/integrations/ui";
import { Button } from "@canonical/react-components";
import React, { FC, FormEvent } from "react";
import { NodeInputProps } from "./helpers";

const getWebAuthnPayload = (evalCode: string): unknown => {
  const tmp = evalCode
    .replace("window.__oryWebAuthnRegistration(", "")
    .replace("window.__oryWebAuthnLogin(", "");
  const raw = tmp.substring(0, tmp.length - 1);
  return JSON.parse(raw) as unknown;
};

interface WebauthnWindow extends Window {
  __oryWebAuthnRegistration: (a: unknown) => void;
  __oryWebAuthnLogin: (a: unknown) => void;
  __oryStartedLogin: boolean;
}

export const NodeInputButton: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  const onClick = attributes.onclick ?? "";

  // automatically start the webauthn login
  const startWebAuthnLogin = () => {
    const webauthnWindow = window as unknown as WebauthnWindow;
    const loginParams = getWebAuthnPayload(onClick);

    if (webauthnWindow.__oryStartedLogin) {
      return;
    }
    webauthnWindow.__oryStartedLogin = true;

    // retry login until the webauthn.js script is finished loading
    // and the __oryWebAuthnLogin function is available
    const triggerWebauthnLogin = (count: number) =>
      setTimeout(() => {
        if ("__oryWebAuthnLogin" in webauthnWindow) {
          webauthnWindow.__oryWebAuthnLogin(loginParams);
        } else if (count < 100) {
          triggerWebauthnLogin(count + 1);
        }
      }, 100);

    triggerWebauthnLogin(0);
  };

  if (onClick?.startsWith("window.__oryWebAuthnLogin(")) {
    startWebAuthnLogin();
  }

  const handleClick = (e: MouseEvent | FormEvent) => {
    // webauthn has a custom handler in webauthn.js to be called
    // preventing the call of eval with below code
    if (onClick?.startsWith("window.__oryWebAuthnRegistration(")) {
      e.stopPropagation();
      e.preventDefault();

      const webauthnWindow = window as unknown as WebauthnWindow;
      const registrationParams = getWebAuthnPayload(onClick);
      webauthnWindow.__oryWebAuthnRegistration(registrationParams);

      return;
    }

    void setValue(attributes.value as string).then(() => dispatchSubmit(e));
  };

  return (
    <>
      <Button
        onClick={handleClick}
        disabled={attributes.disabled || disabled}
        name={attributes.name}
      >
        {getNodeLabel(node)}
      </Button>
    </>
  );
};
