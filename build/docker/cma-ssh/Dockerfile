# Build the manager binary
FROM golang:1.12-stretch as builder

# Copy in the go src
WORKDIR /go/src/github.com/samsung-cnct/cma-ssh
COPY .  ./
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org
RUN go build -o cma-ssh cmd/cma-ssh/main.go

# Copy the controller-manager into a thin image
FROM alpine:3.8
WORKDIR /root/
COPY --from=builder /go/src/github.com/samsung-cnct/cma-ssh/cma-ssh .
ENTRYPOINT ["./cma-ssh"]
