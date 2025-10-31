import type { NextPage } from "next";
import React from "react";
import SetupSecure from "./setup_secure";
import { hasFeatureFlag, useAppConfig } from "../config/useAppConfig";
import { notFound } from "next/navigation";

const ManageSecure: NextPage = () => {
  let appConfig = useAppConfig();
  if (!hasFeatureFlag('totp', appConfig)) {
    notFound()
  }

  return <SetupSecure forceSelfServe={true} />;
};

export default ManageSecure;
