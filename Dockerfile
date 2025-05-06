# syntax = docker/dockerfile:1.12
########################################

FROM --platform=${BUILDPLATFORM} golang:1.24-alpine AS builder
RUN apk update && apk add --no-cache make
WORKDIR /src

COPY go.mod go.sum /src
RUN go mod download && go mod verify

COPY . .
ARG VERSION
RUN make build-all-archs

########################################

FROM --platform=${TARGETARCH} scratch AS release
LABEL org.opencontainers.image.source="https://github.com/siderolabs/talos-cloud-controller-manager" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.description="Talos Cloud Controller Manager"

ARG TARGETARCH
COPY --from=builder /src/talos-cloud-controller-manager-${TARGETARCH} /talos-cloud-controller-manager

ENTRYPOINT ["/talos-cloud-controller-manager"]
