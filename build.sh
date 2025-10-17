#!/bin/sh

# The script requires:
# - rockcraft
# - skopeo with sudo privilege
# - yq
# - docker

set -e

rockcraft pack -v

skopeo --insecure-policy \
  copy "oci-archive:identity-platform-login-ui_$(yq -r '.version' rockcraft.yaml)_amd64.rock" \
  docker-daemon:"$IMAGE"

docker push "$IMAGE"
