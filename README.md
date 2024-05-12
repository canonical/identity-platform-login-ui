# Identity Platform Login UI

[![CI](https://github.com/canonical/identity-platform-login-ui/actions/workflows/ci.yaml/badge.svg)](https://github.com/canonical/identity-platform-login-ui/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/canonical/identity-platform-login-ui/branch/main/graph/badge.svg?token=Aloh6MWghg)](https://codecov.io/gh/canonical/identity-platform-login-ui)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/canonical/identity-platform-login-ui/badge)](https://securityscorecards.dev/viewer/?platform=github.com&org=canonical&repo=identity-platform-login-ui)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196.svg)](https://conventionalcommits.org)

![GitHub Release](https://img.shields.io/github/v/release/canonical/identity-platform-login-ui)
[![Go Reference](https://pkg.go.dev/badge/github.com/canonical/identity-platform-login-ui.svg)](https://pkg.go.dev/github.com/canonical/identity-platform-login-ui)

This is the UI for the Canonical Identity Platform.

## Running the UI

### Build

To create a binary with the UI you need to run:

```shell
make npm-build build
```

Please don't run them in parallel, `build` requires the target `cmd/ui/dist`
which, unless the `js` code has been build independently, requires `npm-build`
If you want to skip the `npm-build` make sure the `js` artifacts are in
the `ui/dist` folder (check the `Makefile` for more advanced information).

This will:

- build the `js` code
- produce a binary called `app` which you can run with:

```shell
PORT=<port number> ./cmd/app
```

### Environment variables

Code dealing with the environment variables resides
in [here](internal/config/specs.go) where each attribute has an annotation which
is the lowercase of the environment variable name.

At the moment the application is sourcing the following from the environment:

- `OTEL_GRPC_ENDPOINT` - needed if we want to use the OTel gRPC exporter for
  traces
- `OTEL_HTTP_ENDPOINT` - needed if we want to use the OTel HTTP exporter for
  traces (if gRPC is specified this gets unused)
- `TRACING_ENABLED` - switch for tracing, defaults to enabled (`true`)
- `LOG_LEVEL` - log level, defaults to `error`
- `LOG_FILE` - log file which the log rotator will write into. default to
  `log.txt`. **Make sure application user has permissions to write**.
- `PORT` - HTTP server port, defaults to `8080`
- `BASE_URL` - the base url that the application will be running on
- `KRATOS_PUBLIC_URL` - address of Kratos APIs
- `HYDRA_ADMIN_URL` - address of Hydra admin APIs
- `OPENFGA_API_SCHEME` - the OpenFGA API scheme
- `OPENFGA_API_HOST` - the OpenFGA API host name
- `OPENFGA_STORE_ID` - the OpenFGA store ID to use
- `OPENFGA_MODEL_ID` - the OpenFGA model ID to use. If not specified, a new
  model will be created

### Container

To build the UI OCI image, you
need [rockcraft](https://canonical-rockcraft.readthedocs-hosted.com). To install
rockcraft run:

```shell
sudo snap install rockcraft --channel=latest/edge --classic
```

To build the image:

```shell
rockcraft pack
```

In order to run the produced image with docker:

```shell
# Import the image to Docker
sudo /snap/rockcraft/current/bin/skopeo --insecure-policy \
    copy oci-archive:./identity-platform-login-ui_0.1_amd64.rock \
    docker-daemon:localhost:32000/identity-platform-login-ui:registry
# Run the image
docker run -d \
  -it \
  --rm \
  -p 8080:8080 \
  --name login-ui \
  localhost:32000/identity-platform-login-ui:registry start login-ui
```

### Development setup

Please install `docker` and `docker-compose`.

You need to have a registered GitHub OAuth application to use for logging in.
To register a GitHub OAuth application:

1) Go to <https://github.com/settings/applications/new>. The application
   name and homepage URL do not matter, but the Authorization callback URL must
   be `http://localhost:4433/self-service/methods/oidc/callback/github`.
2) Generate a client secret
3) Create a file called `.env` on the root of the repository and paste your
   client credentials:

```shell
CLIENT_ID=<client_id>
CLIENT_SECRET=<client_secret>
```

Run the login UI dependencies:

```shell
docker-compose -f docker-compose.dev.yml --build --force-recreate up
```

Build and run the Login UI:

```shell
make build

export KRATOS_PUBLIC_URL=http://localhost:4433
export HYDRA_ADMIN_URL=http://localhost:4445
export BASE_URL=http://localhost:4455
export PORT=4455
export TRACING_ENABLED=false
export LOG_LEVEL=debug
export AUTHORIZATION_ENABLED=false
./app serve
```

To test the authorization code flow you can use the Ory Hydra CLI:

> To install the Ory Hydra CLI follow
> the [instructions](https://www.ory.sh/docs/hydra/self-hosted/install).

```shell
code_client=$(hydra create client \
  --endpoint http://localhost:4445 \
  --name grafana \
  --grant-type authorization_code,refresh_token \
  --response-type code \
  --format json \
  --scope openid,offline_access,email,profile \
  --redirect-uri http://127.0.0.1:4446/callback \
  --audience app_client \
)
hydra perform authorization-code \
  --endpoint http://localhost:4444 \
  --client-id `echo "$code_client" | yq .client_id` \
  --client-secret  `echo "$code_client" | yq .client_secret` \
  --scope openid,profile,email,offline_access
```

### OpenFGA Model Creation

The login UI relies on [OpenFGA](https://github.com/openfga/openfga/) for
authorization decisions.
After you deploy the OpenFGA server, you need to create the OpenFGA store and
model:

```shell
./login-ui-binary create-fga-model --fga-api-token $OPENFGA_API_TOKEN --fga-api-url $OPENFGA_API_URL --store-id $STORE_ID
```

To try it locally you can deploy OpenFGA using docker-compose:

```shell
docker-compose -f docker-compose.dev.yml --build --force-recreate up
```

And run with the store:

```shell
make build

./app create-fga-model --fga-api-token 42 --fga-api-url http://localhost:8080 --store-id 01GP1254CHWJC1MNGVB0WDG1T0

export KRATOS_PUBLIC_URL=http://localhost:4433
export HYDRA_ADMIN_URL=http://localhost:4445
export BASE_URL=http://localhost:4455
export OPENFGA_API_SCHEME=http
export OPENFGA_API_HOST=localhost:8080
export OPENFGA_STORE_ID=01GP1254CHWJC1MNGVB0WDG1T0
export OPENFGA_API_TOKEN=42
export OPENFGA_AUTHORIZATION_MODEL_ID=01HGG9ZQ9PP3P6QHW93QBM55KM
export AUTHORIZATION_ENABLED=true
./app serve
```
