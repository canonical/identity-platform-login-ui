import React, { FC } from "react";
import { Icon } from "@canonical/react-components";
import { PasswordCheckType } from "./Password";
import classNames from "classnames";

type Props = {
  check: PasswordCheckType;
  status: "success" | "error" | "neutral";
};

const PasswordCheck: FC<Props> = ({ check, status }) => {
  const getMessage = () => {
    switch (check) {
      case "lowercase":
        return "Lowercase letters (a-z)";
      case "uppercase":
        return "Uppercase letters (A-Z)";
      case "number":
        return "Number (0-9)";
      case "length":
        return "Minimum 8 characters";
    }
  };

  return (
    <div
      className={classNames({
        "is-success": status === "success",
        "is-error": status === "error",
      })}
    >
      <p
        className={classNames("p-form-validation__message", {
          "is-neutral u-text--muted": status === "neutral",
        })}
      >
        {status === "neutral" && <Icon name="information" />}
        {getMessage()}
      </p>
    </div>
  );
};

export default PasswordCheck;
