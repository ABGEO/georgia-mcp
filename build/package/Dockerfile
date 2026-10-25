# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# The build runs on the platform of the builder, and Go compiles for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.27.2@sha256:820153096fda4de6ac1b7194d12e416b7dd8f4e9a33d17e0345b111b945162d8 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# The whole repository, .git included, so Go writes the commit into the binary.
COPY . .

# The repository belongs to another user in the build stage. Git refuses it, and
# the toolchain then cannot write the commit into the binary.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    git config --global --add safe.directory /src \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /georgia-mcp ./cmd/georgia-mcp

FROM gcr.io/distroless/static-debian13:nonroot@sha256:18f4cc26c9ffa7f20a359298be7deadac5c607c8e03a02b8119f1a5fc2f82c99

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="georgia-mcp" \
  org.opencontainers.image.description="An MCP server for Georgian public services." \
  org.opencontainers.image.source="https://github.com/abgeo/georgia-mcp" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}"

COPY --from=build /georgia-mcp /georgia-mcp

EXPOSE 8080

ENTRYPOINT ["/georgia-mcp"]
CMD ["serve"]
