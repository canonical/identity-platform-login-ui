import React from "react";

export const EmailVerificationPrompt: React.FC<{ email: string }> = ({
  email,
}) => {
  return (
    <p className="u-text--muted">
      An email with a verification code has been sent to {email}. Please check
      your inbox and enter the code below. If the email is not recieved, ensure
      the address is correct.
    </p>
  );
};
