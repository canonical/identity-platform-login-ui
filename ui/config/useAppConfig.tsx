import React, { createContext, useContext, useEffect, useState } from "react";

type FeatureFlags = string | string[];

interface AppConfig {
  oidcSequencingEnabled: boolean;
  baseURL: string;
  identifierFirstEnabled: boolean;
  supportEmail: string;
  flags: FeatureFlags;
}

const defaultAppConfig: AppConfig = {
  oidcSequencingEnabled: false,
  baseURL: "",
  identifierFirstEnabled: false,
  supportEmail: "",
  flags: [],
};

type AppConfigContextValue = AppConfig & { configReady: boolean };

const defaultAppContextValue: AppConfigContextValue = {
  ...defaultAppConfig,
  configReady: false,
};

const AppConfigContext = createContext<AppConfigContextValue>(
  defaultAppContextValue,
);

function useAppConfig() {
  return useContext(AppConfigContext);
}

function hasFeatureFlag(
  requiredFlags: FeatureFlags,
  { flags: activeFlags, configReady }: AppConfigContextValue,
) {
  if (!requiredFlags) {
    return false;
  }

  if (!configReady) {
    return false;
  }

  if (typeof requiredFlags === "string") {
    requiredFlags = [requiredFlags];
  }

  return requiredFlags.every((requiredFlag) =>
    activeFlags.includes(requiredFlag),
  );
}

function AppConfigProvider({
  children,
}: {
  children: React.ReactNode;
}): React.JSX.Element {
  const [contextValue, setContextValue] = useState<AppConfigContextValue>(
    defaultAppContextValue,
  );

  useEffect(() => {
    fetch("../api/v0/app-config", { cache: "no-store" })
      .then((value) => value.json())
      .then((value) => setContextValue({ ...value, configReady: true }))
      .catch(() => setContextValue(defaultAppContextValue));
  }, []);

  return <AppConfigContext value={contextValue}>{children}</AppConfigContext>;
}

export { AppConfigProvider, useAppConfig, hasFeatureFlag };
export type { FeatureFlags };
