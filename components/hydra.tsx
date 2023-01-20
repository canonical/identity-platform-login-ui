import { Configuration, OAuth2Api } from "@ory/client"


const configuration = new Configuration({
  basePath: "http://localhost:4445",
  // accessToken: process.env.HYDRA_ACCESS_TOKEN,
})
const hydraAdmin = new OAuth2Api(
  configuration,
)
export { hydraAdmin }