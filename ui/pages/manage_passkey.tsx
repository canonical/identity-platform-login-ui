import type { NextPage } from "next";
import React from "react";
import SetupPasskey from "./setup_passkey";

const ManagePasskey: NextPage = () => {
  return <SetupPasskey forceSelfServe={true} />;
};

export default ManagePasskey;
