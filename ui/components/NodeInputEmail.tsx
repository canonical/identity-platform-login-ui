import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC, useEffect } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputEmail: FC<NodeInputProps> = ({
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error: upstreamError,
}) => {
  const [hasLocalValidation, setLocalValidation] = React.useState(false);

  const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,6}$/;
  const isInvalid = !emailRegex.test((value as string) ?? "") && value !== "";
  const localError = isInvalid ? "Incorrect email address" : undefined;
  const message = node.messages.map(({ text }) => text).join(" ");
  const defaultValue = message.includes('is not valid "email"')
    ? message.split('"')[1]
    : message;

  const error = hasLocalValidation ? localError : upstreamError;

  useEffect(() => {
    const submitBtn =
      document.getElementsByClassName("p-button--positive")?.[0];
    if (error) {
      submitBtn?.setAttribute("disabled", "");
    } else {
      submitBtn?.removeAttribute("disabled");
    }
  }, [error]);

  if (value == undefined) {
    void setValue(defaultValue);
  }

  const submitOnEnter = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && !error) {
      e.preventDefault();
      e.stopPropagation();
      void dispatchSubmit(e, "code");
    }
  };

  return (
    <Input
      type="email"
      label={getNodeLabel(node)}
      disabled={disabled}
      defaultValue={defaultValue}
      autoFocus={true}
      error={error}
      onChange={(e) => void setValue(e.target.value)}
      onBlur={() => setLocalValidation(true)}
      onKeyDown={submitOnEnter}
    />
  );
};
