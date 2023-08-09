FROM golang:1.21@sha256:a0e3e6859220ee48340c5926794ce87a891a1abb51530573c694317bf8f72543 AS builder

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