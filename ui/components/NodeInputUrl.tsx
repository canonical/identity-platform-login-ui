import { Button } from "@canonical/react-components";
import React, { FC } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputUrl: FC<NodeInputProps> = ({ node }) => {
  return (
    <Button
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();

        // read email from input field
        const email =
          (
            document.querySelector(
              'input[name="identifier"]',
            ) as HTMLInputElement
          )?.value || "";

        // redirect to page with same url as current page, append webauthn query param
        window.location.href = `${window.location.href}&webauthn=true&email=${email}`;
      }}
      className="oidc-login-button"
    >
      <img
        src="logos/Fallback.svg"
        alt="passkey logo"
        style={{ marginRight: "0.5rem" }}
      />
      <span>{node.meta.label?.text}</span>
    </Button>
  );
};
