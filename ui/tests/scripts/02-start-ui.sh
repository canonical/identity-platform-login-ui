#!/bin/bash

set -x
set -e

cd "$(dirname "$0")/../../.."

docker compose up --build -d frontend-ui identity-platform-login-ui

