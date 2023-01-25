import { Configuration, OAuth2Api } from "@ory/client"


const hydraAdmin = new OAuth2Api(
  new Configuration({
    basePath: "/api/hydra",
    baseOptions: {
      withCredentials: true,
    },
  })
)

export { hydraAdmin }