/* eslint-disable */
import type { NextPage } from "next";
import React from "react";
import SetupPasskey from "./setup_passkey";
import { hasFeatureFlag, useAppConfig } from "../config/useAppConfig";
import { notFound } from "next/navigation";

const ManagePasskey: NextPage = () => {
  const appConfig = useAppConfig();
  if (!hasFeatureFlag("webauthn", appConfig)) {
    // notFound()
  }

  return <SetupPasskey forceSelfServe={true} />;
};

export default ManagePasskey;
