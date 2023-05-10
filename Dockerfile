FROM golang:1.19@sha256:9613596d7405705447f36440a59a3a2a1d22384c7568ae1838d0129964c5ba13 AS builder

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