import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC, useCallback, useEffect } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputEmail: FC<NodeInputProps> = ({
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error: upstreamError,
}) => {
  const [error, setError] = React.useState<string | undefined>(upstreamError);

  const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,63}$/;

  const message = node.messages.map(({ text }) => text).join(" ");
  const defaultValue = message.includes('is not valid "email"')
    ? message.split('"')[1]
    : message;

  if (value == undefined) {
    void setValue(defaultValue);
  }

  const getError = useCallback(() => {
    const isInvalid = !emailRegex.test((value as string) ?? "") && value !== "";
    const localError = isInvalid ? "Incorrect email address" : undefined;
    const error = localError ?? upstreamError;
    return error;
  }, [value, upstreamError]);

  const emailValidationOnBlur = useCallback(() => {
    const error = getError();
    setError(error);

    const submitBtn =
      document.getElementsByClassName("p-button--positive")?.[0];
    if (error) {
      submitBtn?.setAttribute("disabled", "");
    } else {
      submitBtn?.removeAttribute("disabled");
    }
  }, [value, upstreamError]);


  useEffect(() => {
    const error = getError();
    if (!error) {
      const submitBtn =
        document.getElementsByClassName("p-button--positive")?.[0];
      submitBtn?.removeAttribute("disabled");
      setError(error);
    }
  }, [upstreamError, value]);

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
      onBlur={emailValidationOnBlur}
      onKeyDown={submitOnEnter}
    />
  );
};
