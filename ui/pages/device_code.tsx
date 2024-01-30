import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";
import { Button, Input } from "@canonical/react-components";

const DeviceCode: NextPage = () => {
  return (
    <PageLayout title="Enter code to continue">
      <Input
        id="code"
        name="code"
        type="text"
        label="Terminal code"
        placeholder="xxxx-xxxx"
      />
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

export default DeviceCode;
