FROM --platform=$BUILDPLATFORM node:24.22.2-alpine@sha256:8a32c8cf649a49915a66e81e7a609223c708d0adabf288ee3f4077d765542f85 AS build

WORKDIR /app

COPY package.json pnpm-lock.yaml ./

RUN --mount=type=cache,target=/pnpm/store \
    corepack enable && pnpm install --frozen-lockfile --ignore-scripts --store-dir /pnpm/store

COPY . .

RUN pnpm build

FROM nginxinc/nginx-unprivileged:1.31.7-alpine@sha256:b5047b9b76845eac124358373c6852377c52e4b05834a47647e955dd91cb8f1e

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="georgia-mcp-website" \
  org.opencontainers.image.description="The website of Georgia MCP." \
  org.opencontainers.image.source="https://github.com/abgeo/georgia-mcp" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}"

COPY .docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
