FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY third_party/ ./third_party/

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/pi-alarm-api ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/pi-alarm-api /usr/local/bin/pi-alarm-api

ENV DATA_DIR=/data

WORKDIR /data

EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/pi-alarm-api"]
