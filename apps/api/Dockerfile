FROM golang:1.24-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /repo/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
COPY apps/api/vendor ./vendor

COPY apps/api ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} go build -mod=vendor -o bin/api .

FROM build AS dirs
RUN mkdir -p /data/avatars

FROM gcr.io/distroless/static-debian12

COPY --from=dirs /data /data
COPY --from=build /repo/apps/api/bin/api /api

EXPOSE 4000
VOLUME ["/data"]

ENTRYPOINT ["/api"]
