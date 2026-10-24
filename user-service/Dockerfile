FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/app ./cmd/user-service

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /bin/app /app
ENTRYPOINT ["/app"]
