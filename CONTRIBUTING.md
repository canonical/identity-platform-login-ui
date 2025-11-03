# Contributing

## Developing

Please install the `pre-commit` to enforce the code conventions and alignment.

```shell
pip install pre-commit
```

Install and update the required pre-commit hooks.

```shell
pre-commit install -t commit-msg
```

### Set up the environment

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
export KRATOS_ADMIN_URL=http://localhost:4434
export HYDRA_ADMIN_URL=http://localhost:4445
export BASE_URL=http://localhost
export PORT=4455
export TRACING_ENABLED=false
export LOG_LEVEL=debug
export AUTHORIZATION_ENABLED=false
export COOKIES_ENCRYPTION_KEY=abcdef01GP1254CHWJC1MNGVB0WDG1T0
export OPENFGA_API_HOST=localhost:8080
export OPENFGA_API_SCHEME=http
export OPENFGA_API_TOKEN=42
export OPENFGA_AUTHORIZATION_MODEL_ID=01HGG9ZQ9PP3P6QHW93QBM55KM # use your authz model ID
export OPENFGA_STORE_ID=01GP1254CHWJC1MNGVB0WDG1T0 # use your store ID
export FEATURE_FLAGS=password,webauthn,backup_codes,totp,account_linking

./app serve
```

To test the authorization code flow you can use the Ory Hydra CLI:

> To install the Ory Hydra CLI follow
> the [instructions](https://www.ory.sh/docs/hydra/self-hosted/install).

```shell
code_client=$(hydra create client \
  --endpoint http://localhost:4445 \
  --name "Some App" \
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
./app create-fga-model --fga-api-token $OPENFGA_API_TOKEN --fga-api-url $OPENFGA_API_URL --store-id $STORE_ID
```

To try it locally you can deploy OpenFGA using docker-compose:

```shell
docker compose -f docker-compose.dev.yml up
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
