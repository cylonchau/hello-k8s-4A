FROM golang:alpine AS builder
MAINTAINER cylon
WORKDIR /4a
COPY ./ /4a
ENV GOPROXY https://goproxy.cn,direct
RUN \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add upx  && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o webhook cmd/hello4A.go && \
    upx -1 webhook && \
    chmod +x webhook

FROM alpine AS runner
WORKDIR /go/4a
COPY --from=builder /4a/webhook .
VOLUME ["/4a"]