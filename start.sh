#!/bin/bash

set -x
set -e

rm -rf cmd/ui/dist

make npm-build build

# Start dependencies
docker compose -f ./docker-compose.dev.yml up  --force-recreate --build --remove-orphans -d

# Start client app
HYDRA_CONTAINER_ID=$(docker ps -aqf "name=identity-platform-login-ui-hydra-1")
HYDRA_IMAGE=ghcr.io/canonical/hydra:2.2.0-canonical

CLIENT_RESULT=$(docker exec "$HYDRA_CONTAINER_ID" \
  hydra create client \
    --endpoint http://127.0.0.1:4445 \
    --name "OIDC App" \
    --grant-type authorization_code,refresh_token,urn:ietf:params:oauth:grant-type:device_code \
    --response-type code \
    --format json \
    --scope openid,profile,offline_access,email \
    --redirect-uri http://127.0.0.1:4446/callback)

CLIENT_ID=$(echo "$CLIENT_RESULT" | cut -d '"' -f4)
CLIENT_SECRET=$(echo "$CLIENT_RESULT" | cut -d '"' -f12)

docker stop oidc_client || true
docker run --network="host" -d --name=oidc_client --rm $HYDRA_IMAGE \
  exec hydra perform authorization-code \
  --endpoint http://localhost:4444 \
  --client-id $CLIENT_ID \
  --client-secret $CLIENT_SECRET \
  --scope openid,profile,email,offline_access \
  --no-open --no-shutdown --format json

# Launch login UI
export KRATOS_PUBLIC_URL="http://localhost:4433"
export KRATOS_ADMIN_URL="http://localhost:4434"
export HYDRA_ADMIN_URL="http://localhost:4445"
export BASE_URL="http://localhost:4455"
export PORT="4455"
export TRACING_ENABLED="false"
export LOG_LEVEL="debug"
export AUTHORIZATION_ENABLED="false"
export COOKIES_ENCRYPTION_KEY=WrfOcYmVBwyduEbKYTUhO4X7XVaOQ1wF
export OIDC_WEBAUTHN_SEQUENCING_ENABLED=false
export MFA_ENABLED=true

go run . serve
