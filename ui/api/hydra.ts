import { Configuration, OAuth2Api } from "@ory/client";

const hydraAdmin = new OAuth2Api(
  new Configuration({
    // Use relative path so that this works when served in a subpath
    basePath: "./api/hydra",
    baseOptions: {
      withCredentials: true,
    },
  })
);

export { hydraAdmin };
