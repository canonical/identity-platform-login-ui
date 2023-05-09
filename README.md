# Identity Platform Login UI

This is the UI for the Canonical Identity Platform.

# Running the UI
## Binary
To create a binary with the UI you need to run:
```console
cd ./ui
npm ci
npm run build
cd ..
go build
```

This will produce a binary called `identity_platform_login_ui` which you can run with:
```console
PORT=1234 ./identity_platform_login_ui &
```
(replace 1234 with an available port of your choice)

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
