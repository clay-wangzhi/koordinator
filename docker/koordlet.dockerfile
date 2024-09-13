FROM --platform=$TARGETPLATFORM golang:1.20 as builder
WORKDIR /go/src/github.com/clay-wangzhi/koordinator

ARG VERSION
ARG TARGETARCH
ENV VERSION $VERSION
ENV GOOS linux
ENV GOARCH $TARGETARCH
ENV GOPROXY="https://goproxy.cn"

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN go build -a -o koordlet cmd/koordlet/main.go


FROM --platform=$TARGETPLATFORM ubuntu:22.04
WORKDIR /
COPY --from=builder /go/src/github.com/clay-wangzhi/koordinator/koordlet .
ENTRYPOINT ["/koordlet"]
