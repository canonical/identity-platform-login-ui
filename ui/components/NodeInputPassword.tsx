import { getNodeLabel } from "@ory/integrations/ui";
import React, { FC, useState, ChangeEvent, KeyboardEvent } from "react";
import { NodeInputProps } from "./helpers";
import PasswordToggle from "./PasswordToggle";

function getLoginStartUrl(): string {
  const url = new URL(window.location.href);
  url.pathname = "/ui/login";
  url.searchParams.delete("flow");
  return url.pathname + url.search;
}

export const NodeInputPassword: FC<NodeInputProps> = ({
  node,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  const [password, setPassword] = useState("");

  const getError = () => {
    const errorMessage = error?.toLowerCase() ?? "";
    if (errorMessage.includes("invalid password")) {
      return "Please enter your password.";
    }

    if (errorMessage.includes("incorrect username or password")) {
      return "Incorrect password. Please try again.";
    }
  };

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    setPassword(e.target.value);
    void setValue(e.target.value);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      e.stopPropagation();
      void dispatchSubmit(e, "password");
    }
  };

  return (
    <PasswordToggle
      tabIndex={2}
      label={
        <>
          <span>{getNodeLabel(node)}</span>
          <a
            href={`./reset_email?return_to=${encodeURIComponent(getLoginStartUrl())}`}
            style={{ float: "right" }}
            tabIndex={3}
          >
            Reset password
          </a>
        </>
      }
      labelClassName="password-label"
      value={password}
      disabled={disabled}
      placeholder="Your Password"
      error={getError()}
      onChange={handleChange}
      onKeyDown={handleKeyDown}
    />
  );
};
