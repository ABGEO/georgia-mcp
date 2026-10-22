# syntax=docker/dockerfile:1

# The build runs on the platform of the builder, and Go compiles for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.27.2 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# The whole repository, so go.work resolves each module of pkg/ before it has a tag.
COPY . .

# The repository belongs to another user in the build stage. Git refuses it, and
# the toolchain then cannot write the commit into the binary.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    git config --global --add safe.directory /src \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /georgia-mcp ./cmd/georgia-mcp

FROM gcr.io/distroless/static-debian13:nonroot

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
