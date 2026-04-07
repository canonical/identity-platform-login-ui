import React, { FC, useCallback, useEffect } from "react";
import PasswordToggle from "./PasswordToggleAlt";
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
  const isMismatch = hasConfirmBlur && password !== confirmation;

  const localValid = hasPassBlur && !isCheckFailed && password === confirmation;
  useEffect(() => {
    if (isValid !== localValid) {
      setValid(localValid);
    }
  }, [isValid, localValid, setValid]);

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
