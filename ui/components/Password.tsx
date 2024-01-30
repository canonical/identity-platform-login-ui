import React, { FC } from "react";
import { Input } from "@canonical/react-components";
import PasswordCheck from "./PasswordCheck";

export type PasswordCheckType = "lowercase" | "uppercase" | "number" | "length";

type Props = {
  checks: PasswordCheckType[];
  password: string;
  setPassword: (password: string) => void;
  isValid: boolean;
  setValid: (isValid: boolean) => void;
  label?: string;
};

const Password: FC<Props> = ({
  checks,
  password,
  setPassword,
  isValid,
  setValid,
  label = "Password",
}) => {
  const [confirmation, setConfirmation] = React.useState("");
  const [hasPassBlur, setPasswordBlurred] = React.useState(false);
  const [hasConfirmBlur, setConfirmationBlurred] = React.useState(false);

  const getStatus = (check: PasswordCheckType) => {
    if (!hasPassBlur) {
      return "neutral";
    }

    switch (check) {
      case "lowercase":
        return /[a-z]/.test(password) ? "success" : "error";
      case "uppercase":
        return /[A-Z]/.test(password) ? "success" : "error";
      case "number":
        return /[0-9]/.test(password) ? "success" : "error";
      case "length":
        return password.length >= 8 ? "success" : "error";
    }
  };

  const isCheckFailed = checks.some((check) => getStatus(check) === "error");
  const isMismatch = hasConfirmBlur && password !== confirmation;

  const localValid = hasPassBlur && !isCheckFailed && password === confirmation;
  if (isValid !== localValid) {
    setValid(localValid);
  }

  return (
    <>
      <Input
        id="password"
        name="password"
        type="password"
        label={label}
        placeholder="Your password"
        onBlur={() => setPasswordBlurred(true)}
        onChange={(e) => setPassword(e.target.value)}
        value={password}
        help={checks.length > 0 && "Password must contain"}
      />
      {checks.map((check) => {
        return (
          <PasswordCheck key={check} check={check} status={getStatus(check)} />
        );
      })}
      <Input
        id="passwordConfirm"
        name="passwordConfirm"
        type="password"
        label={`Confirm ${label}`}
        placeholder="Your password"
        onBlur={() => setConfirmationBlurred(true)}
        onChange={(e) => setConfirmation(e.target.value)}
        error={
          isCheckFailed
            ? "Password does not match requirements"
            : isMismatch
              ? "Passwords do not match"
              : undefined
        }
      />
    </>
  );
};

export default Password;
