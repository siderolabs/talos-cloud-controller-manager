
FROM --platform=${BUILDPLATFORM} golang:1.19-alpine AS builder
RUN apk update && apk add --no-cache make
ENV GO111MODULE on
WORKDIR /src
COPY go.mod go.sum /src
RUN go mod download && go mod verify
COPY . .
RUN make build-all-archs

####

FROM --platform=${BUILDPLATFORM} scratch AS release
ARG ARCH
COPY --from=builder /src/talos-cloud-controller-manager-${ARCH} /talos-cloud-controller-manager

LABEL org.opencontainers.image.source https://github.com/siderolabs/talos-cloud-controller-manager
ENTRYPOINT ["/talos-cloud-controller-manager"]
