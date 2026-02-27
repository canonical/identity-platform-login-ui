import {
  LoginFlow,
  RecoveryFlow,
  SettingsFlow,
  VerificationFlow,
} from "@ory/client";
import { createContext } from "react";

export const FlowContext = createContext<
  LoginFlow | RecoveryFlow | SettingsFlow | VerificationFlow
>({} as LoginFlow | RecoveryFlow | SettingsFlow | VerificationFlow);
