import React, { FC, useEffect, useState, useCallback } from "react";
import PasswordCheck from "./PasswordCheck";
import PasswordToggle from "./PasswordToggle";

export type PasswordCheckType = "lowercase" | "uppercase" | "number" | "length";

type Props = {
  checks: PasswordCheckType[];
  password: string;
  setPassword: (password: string) => void;
  isValid: boolean;
  setValid: (isValid: boolean) => void;
  label?: string;
};

const DEBOUNCE_DURATION = 500;

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const debounceTimer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(debounceTimer);
  }, [value, delay]);
  return debounced;
}

const validateCheck = (check: PasswordCheckType, value: string): boolean => {
  switch (check) {
    case "lowercase":
      return /[a-z]/.test(value);
    case "uppercase":
      return /[A-Z]/.test(value);
    case "number":
      return /\d/.test(value);
    case "length":
      return value.length >= 8;
  }
};

const Password: FC<Props> = ({
  checks,
  password,
  setPassword,
  isValid,
  setValid,
  label = "Password",
}) => {
  const [confirmation, setConfirmation] = useState("");
  const [hasTouched, setHasTouched] = useState(false);
  const [hasBlurred, setHasBlurred] = useState(false);

  const debouncedPassword = useDebounce(password, DEBOUNCE_DURATION);
  const debouncedConfirmation = useDebounce(confirmation, DEBOUNCE_DURATION);

  const getStatus = (check: PasswordCheckType) => {
    if (!hasTouched || !debouncedPassword) return "neutral";
    if (validateCheck(check, debouncedPassword)) return "success";
    return hasBlurred ? "error" : "neutral";
  };

  const isCheckFailed = checks.some((check) => getStatus(check) === "error");
  const isMismatch =
    debouncedConfirmation.length > 0 &&
    debouncedPassword !== debouncedConfirmation;
  const computedValid =
    hasTouched && !isCheckFailed && debouncedPassword === debouncedConfirmation;

  useEffect(() => {
    if (isValid !== computedValid) {
      setValid(computedValid);
    }
  }, [computedValid, setValid]);

  const handlePasswordChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!hasTouched) setHasTouched(true);
      setPassword(e.target.value);
    },
    [hasTouched, setPassword],
  );

  return (
    <>
      <PasswordToggle
        id="password"
        name="password"
        label={label}
        placeholder="Your password"
        onBlur={() => setHasBlurred(true)}
        onChange={handlePasswordChange}
        value={password}
        help={checks.length > 0 && "Password must contain"}
        className={isCheckFailed ? "password-error" : ""}
      />
      <div className="password-checks">
        {checks.map((check) => {
          return (
            <PasswordCheck
              key={check}
              check={check}
              status={getStatus(check)}
            />
          );
        })}
      </div>
      <PasswordToggle
        id="passwordConfirm"
        name="passwordConfirm"
        label={`Confirm ${label}`}
        placeholder="Your password"
        onChange={(e) => setConfirmation(e.target.value)}
        error={
          debouncedConfirmation.length > 0
            ? isCheckFailed
              ? "Password does not match requirements."
              : isMismatch
                ? "Passwords do not match."
                : undefined
            : undefined
        }
        success={
          debouncedConfirmation.length > 0 && !isMismatch && !isCheckFailed
            ? "Passwords match."
            : undefined
        }
      />
    </>
  );
};

export default Password;
