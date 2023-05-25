import { Configuration, FrontendApi } from "@ory/client"

export const kratos = new FrontendApi(  new Configuration({
  // Use relative path so that this works when served in a subpath
  basePath: "./api/kratos",
  baseOptions: {
    withCredentials: true,
  },
})
)
