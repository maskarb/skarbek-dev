
# build the server executable
FROM golang:1.16 as builder

WORKDIR /workspace

COPY vendor/ vendor/
COPY go.mod go.mod
COPY go.sum go.sum

COPY sensor/ sensor/
COPY main.go main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GO111MODULE=on go build -mod vendor -a -o web-server main.go

# Use distroless as minimal base image to package the manager binary
FROM gcr.io/distroless/base:debug-nonroot

WORKDIR /
COPY --from=builder /workspace/web-server .
COPY creds.json .

USER root

EXPOSE 8080/tcp
EXPOSE 8443/tcp

ENTRYPOINT ["/web-server"]
