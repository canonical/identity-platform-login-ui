import React, { useCallback } from "react";
import { Button } from "@canonical/react-components";

const PasskeySequencedSignIn = () => {
  const highlightPasskeyRequirement = useCallback(() => {
    const nameSelector = '*[name="webauthn_register_displayname"]';
    const name = document.querySelector(nameSelector) as HTMLInputElement;

    if (!name?.value) {
      name.classList.add("is-error");
      name.setAttribute("aria-invalid", "true");
      name.setAttribute(
        "aria-errormessage",
        "You must set up a security key to sign in",
      );
    }

    if (!name.nextElementSibling) {
      const warning = document.createElement("p");
      warning.className = "p-form-validation__message";
      warning.textContent = "You must set up a security key to sign in";
      name.after(warning);
    }

    const groupSelector = ".p-form-validation";
    const group = document.querySelectorAll(groupSelector);
    group.forEach((e) => e.classList.add("is-error"));
  }, []);

  return (
    <>
      <hr />
      <div className="u-align--right">
        <Button
          appearance="positive"
          type="button"
          className="u-no-margin--bottom u-align--right"
          onClick={highlightPasskeyRequirement}
        >
          Sign in
        </Button>
      </div>
    </>
  );
};

export default PasskeySequencedSignIn;
