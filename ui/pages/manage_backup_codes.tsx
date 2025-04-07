import type { NextPage } from "next";
import React from "react";
import SetupBackupCodes from "./setup_backup_codes";

const ManageBackupCodes: NextPage = () => {
  return <SetupBackupCodes forceSelfServe={true} />;
};

export default ManageBackupCodes;
