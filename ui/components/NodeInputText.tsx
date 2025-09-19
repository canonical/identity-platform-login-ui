import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC, useEffect, useMemo } from "react";
import { NodeInputProps } from "./helpers";
import { ORY_ERR_ACCOUNT_NOT_FOUND_OR_NO_LOGIN_METHOD } from "../util/constants";

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
  const isIdentifierFirstGroup = node.group === "identifier_first";
  const isEmailInput = node.meta?.label?.text?.toLowerCase?.() === "email";

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

  useEffect(() => {
    if (message) {
      setInputValue(message);
    }
  }, [message, setInputValue]);

  const getError = () => {
    const currentValue = (typeof value === "string" && value) || "";
    if (
      isIdentifierFirstGroup &&
      isEmailInput &&
      attributes.value === currentValue
    ) {
      const serverErrorNode = node.messages.find((m) => m.type === "error");

      if (!serverErrorNode) return undefined;

      const serverErrorId = serverErrorNode?.id;
      const serverErrorText = serverErrorNode?.text ?? "";

      if (currentValue.length === 0) {
        return "Please enter your email address.";
      }

      if (!/^\S+@\S+\.\S+$/.test(currentValue)) {
        return "Enter a valid email address.";
      }

      if (serverErrorId === ORY_ERR_ACCOUNT_NOT_FOUND_OR_NO_LOGIN_METHOD) {
        return "No account found for this email. Verify or create a new account.";
      }

      if (serverErrorText) {
        return serverErrorText;
      }
    }

    if (message.startsWith("Invalid login method")) {
      return "Invalid login method";
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
  };

  const getPlaceholderText = () => {
    if (isIdentifierFirstGroup && isEmailInput) {
      return "Your Email";
    }
    return "";
  };

  return (
    <Input
      type="text"
      autoFocus={true}
      tabIndex={1}
      name={attributes.name}
      label={getNodeLabel(node)}
      disabled={disabled}
      placeholder={getPlaceholderText()}
      value={inputValue}
      error={getError()}
      onChange={(e) => {
        const newValue = e.target.value;
        setInputValue(newValue);
        void setValue(newValue);
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          e.stopPropagation();
          void dispatchSubmit(
            e,
            isIdentifierFirstGroup ? "identifier_first" : "password",
          );
        }
      }}
    />
  );
};
