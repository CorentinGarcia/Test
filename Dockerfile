FROM golang:latest as builder
ADD . /go/src/goapp
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/src/goapp/autoguidage /go/src/goapp/autoguidage.go
FROM alpine:latest
# Install ca-certificates for ssl
RUN set -eux; \
    apk add --no-cache --virtual ca-certificates
ENV INFLUX_DB_NAME="autoguidage"
ENV INFLUX_DB_LOGIN=""
ENV INFLUX_DB_PWD=""
ENV INFLUX_DB_HOST="http://vps198578.ovh.net:8086"
ENV SLACKWEBHOOK_URL="https://hooks.slack.com/services/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"
EXPOSE 9090
WORKDIR /root/
COPY --from=builder /go/src/goapp/autoguidage .
CMD ["./autoguidage"]  