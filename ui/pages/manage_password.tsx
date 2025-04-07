import type { NextPage } from "next";
import React from "react";
import ResetPassword from "./reset_password";

const ManagePassword: NextPage = () => {
  return <ResetPassword forceSelfServe={true} />;
};

export default ManagePassword;
