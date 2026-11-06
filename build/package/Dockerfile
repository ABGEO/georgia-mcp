# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# The build runs on the platform of the builder, and Go compiles for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e AS build

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

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="georgia-mcp" \
  org.opencontainers.image.description="An MCP server for Georgian public services." \
  org.opencontainers.image.source="https://github.com/abgeo/georgia-mcp" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}" \
  io.modelcontextprotocol.server.name="io.github.ABGEO/georgia-mcp"

COPY --from=build /georgia-mcp /georgia-mcp

EXPOSE 8080

ENTRYPOINT ["/georgia-mcp"]
CMD ["serve"]
