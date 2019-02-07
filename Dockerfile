FROM golang:1.11

COPY . /tmp/vsphere-templates-cleaner
RUN cd /tmp/vsphere-templates-cleaner && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -installsuffix cgo

FROM scratch

COPY --from=0 /tmp/vsphere-templates-cleaner/vsphere-templates-cleaner /

ENTRYPOINT ["/vsphere-templates-cleaner"]
