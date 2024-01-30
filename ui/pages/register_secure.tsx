import { Button } from "@canonical/react-components";
import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";

const RegisterSecure: NextPage = () => {
  return (
    <PageLayout title="Secure your account ">
      <p>MFA Setup</p>
      <Button
        type="submit"
        appearance="positive"
        className="u-no-margin--bottom"
      >
        Next
      </Button>
    </PageLayout>
  );
};

export default RegisterSecure;
