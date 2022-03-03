FROM alpine:3.13.2 as build

RUN \
    # install requirements
    apk update && \
    apk add --no-cache --virtual .build-deps \
        ca-certificates \
        make \
        bash \
        wget \
        git \
        curl \
        go \
    # update certs
    update-ca-certificates && \
    # make and install source
    make build && \
    make install && \
    # clear
    make clean && \
    yarn cache clean --all && \
    cd .. && rm -rf opcache-dashboard && \
    apk del .build-deps

FROM alpine:3.13.2 as run

COPY --from=build /usr/local/bin/opcache-dashboard /usr/local/bin/opcache-dashboard

# start service
ENTRYPOINT ["/usr/local/bin/opcache-dashboard"]
