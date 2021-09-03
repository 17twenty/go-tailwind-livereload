# Build stage

FROM golang:1.16.7-buster AS build-env
RUN mkdir -p /go/src/ssr-golang
WORKDIR /go/src/ssr-golang
COPY . .
RUN GO111MODULE=on go mod vendor && env CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build  -o goserver ./cmd/dev-server/dev-server.go

FROM debian:9-slim
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=build-env /go/src/ssr-golang/goserver /app/goserver
COPY ./web /app/web/
WORKDIR /app
EXPOSE 8080
ENTRYPOINT [ "/app/goserver" ]