
# build the server executable
FROM golang:1.17 as builder

WORKDIR /workspace

COPY vendor/ vendor/
COPY go.mod go.mod
COPY go.sum go.sum

COPY internal/ internal/
COPY main.go main.go

RUN CGO_ENABLED=1 GOOS=linux GO111MODULE=on go build -mod vendor -a -o web-server main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/base-debian10:latest-arm64

WORKDIR /opt
COPY --from=builder /workspace/web-server .

EXPOSE 8080/tcp

ENTRYPOINT ["/opt/web-server"]
