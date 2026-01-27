# Login UI

This is a UI for the Ory Kratos identity server. It was based on
the [kratos-selfservice-ui-react-nextjs](https://github.com/ory/kratos-selfservice-ui-react-nextjs/).

## `npm dev`

Run the app in the development mode.

Open <http://localhost:3000> to view it in browser.
Depending on wether you have OpenFGA running or not, port 3000 could be used by it and the frontend app will be running on 3001 instead.

In order to have the local dev env with hmr (hot module replacement) working,
you need to run the npm run dev command with the env var DEV set to true
```shell
DEV=true npm run dev
```

This will enable redirections and rewrites to the correct backend services, in order to mirror the production environment as closely as possible.

The page will reload when you make changes. You may also see any lint errors in
the console.

## `npm test`

Launch the test runner in the interactive watch mode.

See the section
about [running tests](https://facebook.github.io/create-react-app/docs/running-tests)
for more information.

## `npm run build`

Builds the app for production to the `build` folder.

It correctly bundles React in production mode and optimizes the build for the
best performance.

The build is minified and the filenames include the hashes.
Your app is ready to be deployed!

See the section
about [deployment](https://facebook.github.io/create-react-app/docs/deployment)
for more information.

## Testing

We rely on playwright as an executor for end-to-end testing.  To run the tests, follow these steps below.

1. make sure you have installed our test dependencies playwright and oathtool:

    ```console
    sudo npx playwright install-deps
    sudo apt install oathtool
    ```

   Also make sure that you have installed docker:
   See https://docs.docker.com/engine/install/ubuntu/ on how to install docker

2. boot the cluster with dependant backend systems:

    `./ui/tests/scripts/01-start-cluster.sh`

3. start the login ui:

    `./ui/tests/scripts/02-start-ui.sh`

4. register an OIDC client and boot its container using the hydra CLI:

   `./ui/tests/scripts/03-start-oidc-app.sh`

5. Run the tests with the following command:

    `make test-e2e`

You can follow the tests with an open browser. This is helpful in case of failures to debug the root cause.

    `make test-e2e-debug`
