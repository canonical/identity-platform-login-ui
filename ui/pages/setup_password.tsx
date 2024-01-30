import { Button } from "@canonical/react-components";
import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";
import Password from "../components/Password";

const SetupPassword: NextPage = () => {
  const [password, setPassword] = React.useState("");
  const [isPassValid, setPassValid] = React.useState(false);

  return (
    <PageLayout title="Create a password">
      <Password
        checks={["lowercase", "uppercase", "number", "length"]}
        password={password}
        setPassword={setPassword}
        isValid={isPassValid}
        setValid={setPassValid}
      />
      <Button
        type="submit"
        appearance="positive"
        disabled={!isPassValid}
        className="u-no-margin--bottom"
      >
        Next
      </Button>
    </PageLayout>
  );
};

export default SetupPassword;
