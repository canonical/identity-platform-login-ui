import { Configuration, OAuth2Api } from "@ory/hydra-client";

const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "";

const hydraAdmin = new OAuth2Api(
  new Configuration({
    basePath: apiBase ? `${apiBase}/api/hydra` : "../api/hydra",
    baseOptions: {
      withCredentials: true,
    },
  }),
);

export { hydraAdmin };
