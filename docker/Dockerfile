FROM alpine

# set the binary target at build time
ARG BINARY_TARGET

# set up directory structure and add certificates for https/tcp-tls comms
RUN apk update && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/* && \
    mkdir -p /etc/gudgeon && \
    mkdir -p /etc/gudgeon/lists && \
    mkdir -p /var/lib/gudgeon

# copy default config into container and target binary
COPY docker/gudgeon-docker.yml /etc/gudgeon/gudgeon.yml
COPY build/$BINARY_TARGET /bin/gudgeon
COPY docker/docker-entry.sh /entry.sh

# make entrypoint and gudgeon binary executable
RUN chmod +x /entry.sh /bin/gudgeon

# set entry point
ENTRYPOINT /entry.sh

# expose ports (dns, web)
EXPOSE 5354 9009
