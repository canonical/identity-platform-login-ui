# Identity Platform Login UI


[![codecov](https://codecov.io/gh/canonical/identity-platform-login-ui/branch/main/graph/badge.svg?token=Aloh6MWghg)](https://codecov.io/gh/canonical/identity-platform-login-ui)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/canonical/identity-platform-login-ui/badge)](https://securityscorecards.dev/viewer/?platform=github.com&org=canonical&repo=identity-platform-login-ui)
![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/canonical/identity-platform-login-ui)
[![CI](https://github.com/canonical/identity-platform-login-ui/actions/workflows/ci.yaml/badge.svg)](https://github.com/canonical/identity-platform-login-ui/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/canonical/identity-platform-login-ui.svg)](https://pkg.go.dev/github.com/canonical/identity-platform-login-ui)

This is the UI for the Canonical Identity Platform.

# Running the UI


## Build the binary

To create a binary with the UI you need to run:
```console
make npm-build build
```
Please don't run them in parallel, `build` requires the target `cmd/ui/dist` which, unless the `js` code has been build indipendently, requires `npm-build`
If tou wanna skip the `npm-build` make sure the `js` artifcats are is the `ui/dist` folder (check the `Makefile` for more advanced informations)


This will:
* build the `js` code
* produce a binary called `app` which you can run with:

```console
PORT=1234 ./cmd/app
```

(replace 1234 with an available port of your choice)


## Environment variables

Code dealing with the environment variables resides in [here](internal/config/specs.go) where each attribute has an annotation which is the lowercase of the environment variable name.

At the moment the application is sourcing the following from the environment:

* OTEL_GRPC_ENDPOINT - needed if we want to use the otel grpc exporter for traces
* OTEL_HTTP_ENDPOINT - needed if we want to use the otel http exporter for traces (if grpc is specified this gets unused)
* TRACING_ENABLED - switch for tracing, defaults to enabled (`true`)
* LOG_LEVEL - log level, defaults to `error`
* LOG_FILE - log file which the log rotator will write into, *make sure application user has permissions to write*,  defaults to `log.txt`
* PORT - http server port, defaults to `8080`
* BASE_URL - the base url that the application will be running on
* KRATOS_PUBLIC_URL - address of kratos apis
* HYDRA_ADMIN_URL - address of hydra admin apis


## Container
To build the UI oci image, you will need [rockcraft](https://canonical-rockcraft.readthedocs-hosted.com).

To install rockcraft run:
```console
sudo snap install rockcraft --channel=latest/edge --classic
```

To build the image run:
```
rockcraft pack
```

In order to run the produced image with docker run:
```console
# Import the image to Docker
sudo /snap/rockcraft/current/bin/skopeo --insecure-policy copy oci-archive:./identity-platform-login-ui_0.1_amd64.rock docker-daemon:localhost:32000/identity-platform-login-ui:registry
# Run the image
docker run -p 8080:8080 -it --name login-ui --rm localhost:32000/identity-platform-login-ui:registry start login-ui &
```
