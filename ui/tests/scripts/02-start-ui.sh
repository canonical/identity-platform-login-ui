#!/bin/bash

set -x
set -e

cd "$(dirname "$0")/../../.."

rm -rf cmd/ui/dist

make npm-build build

export KRATOS_PUBLIC_URL="http://localhost:4433"
export KRATOS_ADMIN_URL="http://localhost:4434"
export HYDRA_ADMIN_URL="http://localhost:4445"
export BASE_URL="http://localhost"
export PORT="4455"
export TRACING_ENABLED="false"
export LOG_LEVEL="debug"
export AUTHORIZATION_ENABLED="false"
export COOKIES_ENCRYPTION_KEY=WrfOcYmVBwyduEbKYTUhO4X7XVaOQ1wF

go run . serve

