import React, { FC, useState } from "react";
import { Button, Icon } from "@canonical/react-components";
import PasswordInput, { Props } from "./PasswordInput";

const PasswordToggle: FC<Props> = ({ ...props }) => {
  const [isHidden, setIsHidden] = useState(true);

  const togglePasswordVisibility = () => {
    setIsHidden(!isHidden);
  };

  return (
    <div className="password-input-wrapper">
      <PasswordInput {...props} type={isHidden ? "password" : "text"} />
      <Button
        type="button"
        appearance="base"
        className="password-visibility"
        aria-label={isHidden ? "Show password" : "Hide password"}
        hasIcon
        onClick={togglePasswordVisibility}
      >
        <Icon name={isHidden ? "show" : "hide"} />
      </Button>
    </div>
  );
};

export default PasswordToggle;
