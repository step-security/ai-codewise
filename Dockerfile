FROM golang:1.20@sha256:685a22e459f9516f27d975c5cc6accc11223ee81fdfbbae60e39cc3b87357306 AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN apt-get -qq update && \
    apt-get -yqq install upx

WORKDIR /
COPY . .
ARG API_ENDPOINT
RUN echo "API Endpoint: $API_ENDPOINT"
RUN go build -ldflags "-X main.APIEndpoint=$API_ENDPOINT" \
    -a \
    -o /bin/app \
    . \
    && strip /bin/app \
    && upx -q -9 /bin/app

RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc_passwd /etc/passwd
COPY --from=builder --chown=65534:0 /bin/app /app

USER nobody
ENTRYPOINT ["/app"]