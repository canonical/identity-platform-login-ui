import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";

const DeviceComplete: NextPage = () => {
  return (
    <PageLayout title="Sign in successful">
      <p>
        Your device has been successfully connected. You can close this tab.
      </p>
    </PageLayout>
  );
};

export default DeviceComplete;
