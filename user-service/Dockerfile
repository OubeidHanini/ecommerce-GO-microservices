# Build context: repository root (needed for the shared platform module)
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY platform/ platform/
COPY user-service/go.mod user-service/go.sum user-service/
WORKDIR /src/user-service
RUN go mod download
COPY user-service/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/app ./cmd/user-service

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /bin/app /app
ENTRYPOINT ["/app"]
