import { Configuration, FrontendApi } from "@ory/client"
import { edgeConfig } from "@ory/integrations/next"

export const ory = new FrontendApi(new Configuration(edgeConfig))