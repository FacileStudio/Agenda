FROM oven/bun:1 AS client-build
WORKDIR /app
COPY apps/client/package.json apps/client/bun.lock* ./
RUN bun install --frozen-lockfile
COPY apps/client/ .
RUN bun run build

FROM golang:1.26-alpine AS api-build
WORKDIR /app
ARG APP_VERSION=dev
COPY apps/api/ .
RUN go build -mod=vendor -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /agenda .

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=api-build /agenda /agenda
COPY --from=client-build /app/build /client
ENV CLIENT_DIR=/client
EXPOSE 4000
ENTRYPOINT ["/agenda"]
