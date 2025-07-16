import { Configuration, FrontendApi } from "@ory/client";

export const kratos = new FrontendApi(
  new Configuration({
    // WIP needs to be configurable
    basePath: "..",
    baseOptions: {
      withCredentials: true,
    },
  })
);
