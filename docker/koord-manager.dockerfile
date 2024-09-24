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

COPY apis/ apis/
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 go build -a -o koord-manager cmd/koord-manager/main.go

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /go/src/github.com/clay-wangzhi/koordinator/koord-manager .
ENTRYPOINT ["/koord-manager"]
