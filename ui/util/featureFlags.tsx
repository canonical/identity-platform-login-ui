import { FeatureFlags, hasFeatureFlag, useAppConfig } from "../config/useAppConfig";
import React from "react";

type FeatureEnabledProps = {
  flags: FeatureFlags
  children: React.ReactNode
};


export const FeatureEnabled = ({flags: requiredFlags, children}: FeatureEnabledProps): React.JSX.Element => {
  const appConfig = useAppConfig();
  return hasFeatureFlag(requiredFlags, appConfig) ? <>{children}</> : <></>
}
