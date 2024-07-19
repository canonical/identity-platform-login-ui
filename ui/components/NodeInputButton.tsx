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

  // automatically start the webauthn login
  if (
    onClick?.startsWith("window.__oryWebAuthnLogin(") &&
    getNodeLabel(node) === "Continue"
  ) {
    startWebAuthnLogin();
  }

  const handleClick = (e: MouseEvent | FormEvent) => {
    // webauthn has a custom handler in webauthn.js to be called
    // preventing the call of eval with below code
    if (onClick?.startsWith("window.__oryWebAuthnRegistration(")) {
      e.stopPropagation();
      e.preventDefault();

      const nameSelector = '*[name="webauthn_register_displayname"]';
      const name = document.querySelector(nameSelector) as HTMLInputElement;

      if (!name?.value) {
        name.classList.add("is-error");
        name.setAttribute("aria-invalid", "true");
        name.setAttribute("aria-errormessage", "Required field");

        if (!name.nextElementSibling) {
          const warning = document.createElement("p");
          warning.className = "p-form-validation__message";
          warning.textContent = "Required field";
          name.after(warning);
        }

        const groupSelector = ".p-form-validation";
        const group = document.querySelectorAll(groupSelector);
        group.forEach((e) => e.classList.add("is-error"));

        return;
      }

      const webauthnWindow = window as unknown as WebauthnWindow;
      const registrationParams = getWebAuthnPayload(onClick);
      webauthnWindow.__oryWebAuthnRegistration(registrationParams);

      return;
    }

    if (onClick?.startsWith("window.__oryWebAuthnLogin(")) {
      e.stopPropagation();
      e.preventDefault();

      const webauthnWindow = window as unknown as WebauthnWindow;
      const loginParams = getWebAuthnPayload(onClick);
      webauthnWindow.__oryWebAuthnLogin(loginParams);

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
