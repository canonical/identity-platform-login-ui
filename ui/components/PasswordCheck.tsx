import React, { FC } from "react";
import { Icon } from "@canonical/react-components";
import { PasswordCheckType } from "./Password";

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

  switch (status) {
    case "success":
      return (
        <div className="p-form-validation is-success">
          <p className="p-form-validation__message">{getMessage()}</p>
        </div>
      );
    case "error":
      return (
        <div className="p-form-validation is-error">
          <p className="p-form-validation__message">{getMessage()}</p>
        </div>
      );
    case "neutral":
      return (
        <p className="p-text--small u-text--muted">
          <Icon name="information" />
          &nbsp;&nbsp;{getMessage()}
        </p>
      );
  }
};

export default PasswordCheck;
