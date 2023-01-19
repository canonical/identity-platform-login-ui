import { Configuration, FrontendApi } from "@ory/client"
import { edgeConfig } from "@ory/integrations/next"

export const kratos = new FrontendApi(new Configuration({
    basePath: process.env.ORY_KRATOS_URL
}))