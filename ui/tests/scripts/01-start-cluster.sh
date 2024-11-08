#!/bin/bash

set -x
set -e

cd "$(dirname "$0")/../../.."

docker compose -f docker-compose.dev.yml up --force-recreate --remove-orphans
