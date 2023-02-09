# Getting Started with Create React App

This is a UI for the Ory Kratos identity server. It was based on the [kratos-selfservice-ui-react-nextjs](https://github.com/ory/kratos-selfservice-ui-react-nextjs/).

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

<!-- TODO: Change the name once we move repositories -->
This will produce a binary called `ory_ui` which you can run with:
```console
PORT=some-port ./ory_ui
```

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
sudo /snap/rockcraft/current/bin/skopeo --insecure-policy copy oci-archive:./my-rock-name_0.1_amd64.rock docker-daemon:identity-platform-ui:1.0
# Run the image
docker run identity-platform-ui:1.0
```


