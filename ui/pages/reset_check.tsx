import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";

const ResetCheck: NextPage = () => {
  return (
    <PageLayout title="Check your email: ">
      <p className="u-text--muted">
        Follow the instructions to reset your password.
      </p>
    </PageLayout>
  );
};

export default ResetCheck;
