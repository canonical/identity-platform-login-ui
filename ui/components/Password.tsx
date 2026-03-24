import React, { FC, useCallback } from "react";
import PasswordToggle from "./PasswordToggle";
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
  const [isEditingPass, setIsEditingPass] = React.useState(true);

  const getStatus = useCallback(
    (check: PasswordCheckType) => {
      switch (check) {
        case "lowercase":
          return /[a-z]/.test(password)
            ? "success"
            : isEditingPass
              ? "neutral"
              : "error";
        case "uppercase":
          return /[A-Z]/.test(password)
            ? "success"
            : isEditingPass
              ? "neutral"
              : "error";
        case "number":
          return /[0-9]/.test(password)
            ? "success"
            : isEditingPass
              ? "neutral"
              : "error";
        case "length":
          return password.length >= 8
            ? "success"
            : isEditingPass
              ? "neutral"
              : "error";
      }
    },
    [password, isEditingPass],
  );

  const isCheckFailed = checks.some((check) => getStatus(check) === "error");
  const isMismatch = hasConfirmBlur && password !== confirmation;

  const localValid = hasPassBlur && !isCheckFailed && password === confirmation;
  if (isValid !== localValid) {
    setValid(localValid);
  }

  return (
    <>
      <PasswordToggle
        id="password"
        name="password"
        label={label}
        placeholder="Your password"
        onBlur={() => {
          setPasswordBlurred(true);
          setIsEditingPass(false);
        }}
        onFocus={() => setIsEditingPass(true)}
        onChange={(e) => setPassword(e.target.value)}
        value={password}
        help={checks.length > 0 && "Password must contain"}
        error={
          isCheckFailed ? "Password does not match requirements" : undefined
        }
      />
      <div className="password-checks">
        {checks.map((check) => {
          const status = getStatus(check);
          return <PasswordCheck key={check} check={check} status={status} />;
        })}
      </div>
      <PasswordToggle
        id="passwordConfirm"
        name="passwordConfirm"
        label={`Confirm ${label}`}
        placeholder="Your password"
        onBlur={() => setConfirmationBlurred(true)}
        onChange={(e) => setConfirmation(e.target.value)}
        error={isMismatch ? "Passwords do not match" : undefined}
      />
    </>
  );
};

export default Password;
