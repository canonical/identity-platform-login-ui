#!/bin/bash

set -x
set -e

cd "$(dirname "$0")/../../.."

pwd

export KRATOS_PUBLIC_URL="http://localhost:4433"
export KRATOS_ADMIN_URL="http://localhost:4434"
export HYDRA_ADMIN_URL="http://localhost:4445"
export BASE_URL="http://localhost:4455"
export PORT="4455"
export TRACING_ENABLED="false"
export LOG_LEVEL="debug"
export AUTHORIZATION_ENABLED="false"

docker-compose -f docker-compose.dev.yml up --build --force-recreate --remove-orphans
