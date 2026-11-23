# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# The build runs on the platform of the builder, and Go compiles for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS build

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

FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

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
