# Kala's container image: a multi-stage build on pinned base images. The
# runtime image keeps a shell and common tools, because jobs run commands,
# and runs Kala as a non-root user.
#
# Build it with the version, commit and build date, for example:
#   docker build --build-arg VERSION=v2.0.0 --build-arg COMMIT=$(git rev-parse HEAD) \
#     --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) -t kala .

# golang:1.27.4-trixie
FROM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS build
# npm installs the web interface's libraries (make webui-libs).
RUN apt-get update \
    && apt-get install -y --no-install-recommends nodejs npm \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 make bin/kala VERSION="$VERSION" COMMIT="$COMMIT" BUILD_DATE="$BUILD_DATE"

# debian:trixie-slim
FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2
RUN apt-get update \
    && apt-get install -y --no-install-recommends bash ca-certificates curl procps tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 kala \
    && useradd --system --uid 10001 --gid kala --create-home --home-dir /var/lib/kala kala
COPY --from=build /src/bin/kala /usr/local/bin/kala
USER 10001:10001
# The default bolt database file is created in the working directory. To keep
# it, mount a volume there that uid 10001 can write to.
WORKDIR /var/lib/kala
EXPOSE 8000
ENTRYPOINT ["kala"]
CMD ["serve"]
