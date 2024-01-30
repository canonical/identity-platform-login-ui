import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";

const ResetComplete: NextPage = () => {
  return (
    <PageLayout title="Password reset complete">
      <p className="u-text--muted">You can close this tab.</p>
    </PageLayout>
  );
};

export default ResetComplete;
