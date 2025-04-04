import type { NextPage } from "next";
import React from "react";
import SetupSecure from "./setup_secure";

const ManageSecure: NextPage = () => {
  return <SetupSecure forceSelfServe={true} />;
};

export default ManageSecure;
