import React, { FC, useState } from "react";
import { Input, Button, Icon } from "@canonical/react-components";

type Props = Omit<React.ComponentProps<typeof Input>, "type">;

const PasswordToggle: FC<Props> = ({ ...props }) => {
  const [isHidden, setIsHidden] = useState(true);

  const togglePasswordVisibility = () => {
    setIsHidden(!isHidden);
  };

  return (
    <div className="password-input-wrapper">
      <Input {...props} type={isHidden ? "password" : "text"} />
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
