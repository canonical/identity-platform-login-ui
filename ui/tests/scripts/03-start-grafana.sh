#!/bin/bash

set -x
set -e

cd "$(dirname "$0")/../../.."

GRAFANA_CONTAINER_ID=$(docker ps -a | grep grafana | awk '{print $1}')
if [ -n "${GRAFANA_CONTAINER_ID}" ]
then
  docker stop "$GRAFANA_CONTAINER_ID"

  docker rm "$GRAFANA_CONTAINER_ID"
fi

HYDRA_CONTAINER_ID=$(docker ps | grep hydra | awk '{print $1}')

CLIENT_RESULT=$(docker exec "$HYDRA_CONTAINER_ID" \
  hydra create client \
    --endpoint http://127.0.0.1:4445 \
    --name grafana \
    --grant-type authorization_code,refresh_token \
    --response-type code,id_token \
    --format json \
    --scope openid,offline_access,email \
    --redirect-uri http://localhost:2345/login/generic_oauth)

CLIENT_ID=$(echo "$CLIENT_RESULT" | cut -d '"' -f4)
CLIENT_SECRET=$(echo "$CLIENT_RESULT" | cut -d '"' -f12)

docker run -d --name=grafana -p 2345:2345 --network identity-platform-login-ui_intranet \
-e "GF_SERVER_HTTP_PORT=2345" \
-e "GF_AUTH_GENERIC_OAUTH_ENABLED=true" \
-e "GF_AUTH_GENERIC_OAUTH_AUTH_ALLOWED_DOMAINS=hydra,localhost" \
-e "GF_AUTH_GENERIC_OAUTH_NAME=Identity Platform" \
-e "GF_AUTH_GENERIC_OAUTH_CLIENT_ID=$CLIENT_ID" \
-e "GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET=$CLIENT_SECRET" \
-e "GF_AUTH_GENERIC_OAUTH_SCOPES=openid offline_access email" \
-e "GF_AUTH_GENERIC_OAUTH_AUTH_URL=http://localhost:4444/oauth2/auth" \
-e "GF_AUTH_GENERIC_OAUTH_TOKEN_URL=http://hydra:4444/oauth2/token" \
-e "GF_AUTH_GENERIC_OAUTH_API_URL=http://hydra:4444/userinfo" \
grafana/grafana

echo ""
echo "Grafana started."
