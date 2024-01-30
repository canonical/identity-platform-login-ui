import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";

const SetupComplete: NextPage = () => {
  return (
    <PageLayout title="Account setup complete">
      <p className="u-text--muted">You can close this tab.</p>
    </PageLayout>
  );
};

export default SetupComplete;
