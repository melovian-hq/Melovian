# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
# syntax=docker/dockerfile:1

FROM golang:1.26-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS backend
WORKDIR /src
ARG VERSION=dev
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN test -d frontend/dist || (echo "missing frontend/dist: run task common:build:frontend first" && exit 1)
RUN test -d vendor || (echo "missing vendor/: run task go:vendor" && exit 1)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -tags server -trimpath \
  -ldflags="-s -w -X melovian/internal/compat.Version=${VERSION}" \
  -o /out/melovian .

FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40
ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED=
ARG SOURCE=
ARG TITLE=Melovian
ARG DESCRIPTION="Music player for local libraries and Subsonic-compatible servers"
ARG VENDOR="Quad4 Software"
ARG AUTHORS=Quad4
ARG LICENSES=Apache-2.0
LABEL org.opencontainers.image.title="${TITLE}" \
      org.opencontainers.image.description="${DESCRIPTION}" \
      org.opencontainers.image.vendor="${VENDOR}" \
      org.opencontainers.image.authors="${AUTHORS}" \
      org.opencontainers.image.licenses="${LICENSES}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.source="${SOURCE}" \
      org.opencontainers.image.url="${SOURCE}"
RUN apk add --no-cache ca-certificates tini wget \
  && addgroup -S melovian -g 1000 \
  && adduser -S melovian -G melovian -u 1000 -h /data
WORKDIR /app
COPY --from=backend --chown=melovian:melovian /out/melovian /app/melovian
USER melovian
ENV MELOVIAN_DATA=/data \
    MELOVIAN_LISTEN=0.0.0.0:8080 \
    CACHE_ENABLED=true
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/melovian"]
