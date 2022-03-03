FROM alpine:3.13.2 as build

RUN \
    # install requirements
    apk update && \
    apk add --no-cache \
        ca-certificates \
        make \
        bash \
        wget \
        git \
        curl \
        go && \
    # make and install source
    make build && make install

FROM alpine:3.13.2 as run

COPY --from=build /usr/local/bin/ltt /usr/local/bin/ltt

# start service
ENTRYPOINT ["/usr/local/bin/ltt"]
