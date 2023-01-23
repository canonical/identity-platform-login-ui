// @ory/integrations offers a package for integrating with NextJS.
import { config, createApiHandler } from "@ory/integrations/next-edge"

// We need to export the config.
export { config }

export default createApiHandler({
  // Because vercel.app is a public suffix and setting cookies for
  // vercel.app is not possible.
  dontUseTldForCookieDomain: true,
  fallbackToPlayground: false,
  apiBaseUrlOverride: process.env.HYDRA_ADMIN_URL,
})
