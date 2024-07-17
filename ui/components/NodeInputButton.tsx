import { getNodeLabel } from "@ory/integrations/ui";
import { Button } from "@canonical/react-components";
import React, { FC, FormEvent } from "react";
import { NodeInputProps } from "./helpers";

const getWebAuthnRegistrationPayload = (evalCode: string): unknown => {
  const tmp = evalCode.replace("window.__oryWebAuthnRegistration(", "");
  const raw = tmp.substring(0, tmp.length - 1);
  return JSON.parse(raw) as unknown;
};

interface WebauthnWindow extends Window {
  __oryWebAuthnRegistration: (a: unknown) => void;
}

export const NodeInputButton: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  const handleClick = (e: MouseEvent | FormEvent) => {
    const onClick = attributes.onclick;

    // webauthn registration has a custom handler in webauthn.js to be called
    // preventing the call of eval with below code
    if (onClick?.startsWith("window.__oryWebAuthnRegistration(")) {
      e.stopPropagation();
      e.preventDefault();

      const webauthnWindow = window as unknown as WebauthnWindow;
      const registrationParams = getWebAuthnRegistrationPayload(onClick);
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
