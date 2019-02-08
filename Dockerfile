FROM golang:1.11

COPY . /tmp/vsphere-templates-cleaner
RUN cd /tmp/vsphere-templates-cleaner && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -installsuffix cgo

FROM alpine:3.9

RUN apk add --no-cache ca-certificates
RUN addgroup -g 1000 -S cleaner && \
    adduser -u 1000 -S cleaner -G cleaner
USER cleaner

COPY --from=0 /tmp/vsphere-templates-cleaner/vsphere-templates-cleaner /

ENTRYPOINT ["/vsphere-templates-cleaner"]
CMD ["--help"]
