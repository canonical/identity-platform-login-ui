import { SettingsFlow } from "@ory/client";

export const formatReturnTo = (
  returnTo: string | string[] | undefined,
  forceSelfServe?: boolean,
) => {
  if (forceSelfServe) {
    const origin = window.location.origin;
    const path = window.location.pathname.replace("setup", "manage");
    return `${origin}${path}`;
  }

  return returnTo ? String(returnTo) : undefined;
};

export const hasSelfServeReturn = (flow?: SettingsFlow) => {
  return flow?.return_to?.includes("/manage_") ?? false;
};

export type IdentityTraits = {
  name?: string;
  surname?: string;
  identity?: string;
  email?: string;
} | null;

export const getLoggedInName = (flow?: SettingsFlow): string => {
  const traits = flow?.identity?.traits as IdentityTraits;
  if (!traits) {
    return "";
  }

  if (traits.name && traits.surname) {
    return traits.name + " " + traits.surname;
  }

  if (traits.identity) {
    return traits.identity;
  }

  if (traits.email) {
    return traits.email;
  }

  return "";
};

export const getFullName = (flow?: SettingsFlow): string => {
  const traits = flow?.identity?.traits as IdentityTraits;
  if (!traits) {
    return "";
  }

  if (traits.name && traits.surname) {
    return traits.name + " " + traits.surname;
  }

  return "";
};
