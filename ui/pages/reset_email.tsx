import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";
import { Button, Input } from "@canonical/react-components";

const ResetComplete: NextPage = () => {
  return (
    <PageLayout title="Enter an email to reset your password">
      <Input
        id="email"
        name="email"
        type="text"
        label="Email"
        placeholder="Your email"
      />
      <Button type="button" className="u-no-margin--bottom">
        Back
      </Button>
      <Button
        type="submit"
        appearance="positive"
        className="u-no-margin--bottom"
      >
        Reset password
      </Button>
    </PageLayout>
  );
};

export default ResetComplete;
