import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { Component, FC, useEffect, useMemo } from "react";
import { NodeInputProps } from "./helpers";
import CountDown from "./CountDown";

export const NodeInputText: FC<NodeInputProps> = ({
  attributes,
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  const [inputValue, setInputValue] = React.useState(
    attributes.value as string,
  );

  const urlParams = new URLSearchParams(window.location.search);
  const isWebauthn = urlParams.get("webauthn") === "true";
  const ucFirst = (s?: string) =>
    s ? String(s[0]).toUpperCase() + String(s).slice(1) : s;

  let deduplicateValues: string[] = [];
  if (node.meta.label?.context) {
    if ("deduplicateValues" in node.meta.label.context) {
      deduplicateValues = node.meta.label.context.deduplicateValues as string[];
    }
  }
  const isDuplicate = deduplicateValues.includes(value as string);

  const message = useMemo(
    () => node.messages.map(({ text }) => text).join(" "),
    [node.messages],
  );

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

  useEffect(() => {
    if (node.messages.length === 0) {
      return;
    }
    for (const msg of node.messages) {
      if (msg.type !== "info") {
        return;
      }
    }
    if (message) {
      setInputValue(message);
    }
  }, [message, setInputValue]);

  const getError = useMemo(() => {
    if (message.startsWith("Invalid login method")) {
      return "Invalid login method";
    }

    if (node.messages.length > 0 && node.messages[0].type === "error") {
      return message;
    }

    if (isDuplicate) {
      return "This value is already in use";
    }

    if (
      attributes.name === "code" ||
      attributes.name === "totp" ||
      attributes.name === "totp_code" ||
      attributes.name === "lookup_secret" ||
      (isWebauthn && attributes.name === "identifier")
    ) {
      return ucFirst(error);
    }

    return undefined;
  }, [message, node.messages, isDuplicate, attributes.name, isWebauthn, error]);

  const getSuccess = useMemo(() => {
    if (node.messages.length > 0 && node.messages[0].type === "success") {
      return (
        <CountDown
          initialSeconds={10}
          wrapperText="Code sent. You can request again in 00:"
        />
      );
    }
    return undefined;
  }, [node.messages]);

  return (
    <>
      {beforeComponent}
      <Input
        type="text"
        autoFocus={true}
        tabIndex={1}
        name={attributes.name}
        label={getNodeLabel(node)}
        disabled={disabled}
        value={inputValue}
        success={getSuccess}
        error={getError}
        onChange={(e) => {
          const newValue = e.target.value;
          setInputValue(newValue);
          void setValue(newValue);
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            e.stopPropagation();
            void dispatchSubmit(e, "password");
          }
        }}
      />
      {afterComponent}
    </>
  );
};
