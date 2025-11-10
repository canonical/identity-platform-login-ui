import type { NextPage } from "next";
import React from "react";
import ResetPassword from "./reset_password";
import {
  hasFeatureFlag,
  useAppConfig,
} from "../config/useAppConfig";
import { notFound } from "next/navigation";

const ManagePassword: NextPage = () => {
  const appConfig = useAppConfig();
  if (!hasFeatureFlag("password", appConfig)) {
    notFound()
  }

  return <ResetPassword forceSelfServe={true}/>;
};

export default ManagePassword;
