import React, { FC, InputHTMLAttributes, ReactNode, useState } from "react";
import { Button, Icon } from "@canonical/react-components";
import PasswordInput from "./PasswordInput";

type PropsWithSpread<P, H> = P & Omit<H, keyof P>;
type Props = PropsWithSpread<
  {
    /**
     * The content for caution validation.
     */
    caution?: ReactNode;
    /**
     * Optional class(es) to pass to the input element.
     */
    className?: string | null;
    /**
     * The content for error validation message. Controls the value of aria-invalid attribute.
     */
    error?: ReactNode;
    /**
     * Help text to show below the field.
     */
    help?: ReactNode;
    /**
     * Optional class(es) to pass to the help text element.
     */
    helpClassName?: string;
    /**
     * The id of the input.
     */
    id?: string;
    /**
     * The label for the field.
     */
    label?: ReactNode;
    /**
     * Optional class(es) to pass to the label component.
     */
    labelClassName?: string;
    /**
     * Whether the field is required.
     */
    required?: boolean;
    /**
     * Whether the form field should have a stacked appearance.
     */
    stacked?: boolean;
    /**
     * The content for success validation.
     */
    success?: ReactNode;
    /**
     * Whether to focus on the input on initial render.
     */
    takeFocus?: boolean;
    /**
     * Delay takeFocus in milliseconds i.e. to let animations finish
     */
    takeFocusDelay?: number;
    /**
     * Optional class(es) to pass to the wrapping Field component
     */
    wrapperClassName?: string;
  },
  InputHTMLAttributes<HTMLInputElement>
>;
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
