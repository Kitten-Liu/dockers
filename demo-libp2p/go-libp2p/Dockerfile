FROM golang:1.22.2-alpine3.19 AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o golibp2p .

FROM alpine:latest AS app

RUN sed 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' -i /etc/apk/repositories && apk add tcpdump
COPY --from=builder /build/golibp2p /
CMD ["/golibp2p"]
