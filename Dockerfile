# One Dockerfile for every service: only the service name changes.
# Build context: repository root (needed for the shared platform module).
FROM golang:1.27-alpine AS build
ARG SERVICE
WORKDIR /src
COPY platform/ platform/
COPY ${SERVICE}/go.* ${SERVICE}/
WORKDIR /src/${SERVICE}
RUN go mod download
COPY ${SERVICE}/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/app ./cmd/${SERVICE}

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /bin/app /app
ENTRYPOINT ["/app"]
