import type { NextPage } from "next";
import React from "react";
import SetupBackupCodes from "./setup_backup_codes";
import { hasFeatureFlag, useAppConfig } from "../config/useAppConfig";
import { notFound } from "next/navigation";

const ManageBackupCodes: NextPage = () => {
  const appConfig = useAppConfig();
  if (!hasFeatureFlag("backup_codes", appConfig)) {
    notFound()
  }

  return <SetupBackupCodes forceSelfServe={true} />;
};

export default ManageBackupCodes;
