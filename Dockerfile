ARG GO_VERSION=1.19
ARG NODE_VERSION=19

FROM node:${NODE_VERSION}-bullseye as node-builder
WORKDIR /app
COPY ui/package.json ui/package-lock.json ./
RUN npm ci --frozen-lockfile
COPY ui/ .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM golang:${GO_VERSION}-bullseye AS go-builder
WORKDIR /app
COPY go.mod main.go ./
COPY --from=node-builder /app/dist ./ui/dist
RUN go build .

FROM public.ecr.aws/lts/ubuntu:22.04
WORKDIR /app
COPY --from=go-builder /app/ory_ui .

ENTRYPOINT ["./ory_ui"]

EXPOSE 8080