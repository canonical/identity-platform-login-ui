# Login UI

This is a UI for the Ory Kratos identity server. It was based on
the [kratos-selfservice-ui-react-nextjs](https://github.com/ory/kratos-selfservice-ui-react-nextjs/).

## `npm dev`

Run the app in the development mode.

Open <http://localhost:3000> to view it in browser.

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

1. boot the cluster with dependant backend systems:

    `./ui/tests/scripts/01-start-cluster.sh`

2. start the login ui:

    `./ui/tests/scripts/02-start-ui.sh`

3. register grafana as client and boot its container:

   `./ui/tests/scripts/03-start-grafana.sh`

4. Run the tests with the following command:

    `cd ui && npx playwright test`

You can follow the tests with the ui parameter. This is helpful in case of failures to debug the root cause.

    cd ui && npx playwright test --ui
