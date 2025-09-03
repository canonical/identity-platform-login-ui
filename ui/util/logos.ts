export const getProviderImage = (value: string) => {
  if (value.toLowerCase().startsWith("auth0")) {
    return "logos/Auth0.svg";
  }
  if (value.toLowerCase().startsWith("github")) {
    return "logos/Github.svg";
  }
  if (value.toLowerCase().startsWith("google")) {
    return "logos/Google.svg";
  }
  if (value.toLowerCase().startsWith("microsoft")) {
    return "logos/Microsoft.svg";
  }
  if (value.toLowerCase().startsWith("ping")) {
    return "logos/Ping.svg";
  }
  return "logos/Fallback.svg";
};
