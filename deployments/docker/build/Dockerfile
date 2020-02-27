# Build Container Image used to Pull Files
FROM alpine as build
ARG tag
ENV REL_TAG=$tag

RUN apk -U upgrade && \
    apk add curl unzip

RUN curl -sL -o /tmp/dsiem-server_linux_amd64.zip https://github.com/defenxor/dsiem/releases/download/$REL_TAG/dsiem-server_linux_amd64.zip && \
    unzip /tmp/dsiem-server_linux_amd64.zip -d /dsiem && \
    curl -sL -o /tmp/s6-overlay-amd64.tar.gz https://github.com/just-containers/s6-overlay/releases/download/v1.22.1.0/s6-overlay-amd64.tar.gz && \
    mkdir /s6 && \
    tar xzf /tmp/s6-overlay-amd64.tar.gz -C /s6

# Final Container Image
FROM alpine

# Install packages
RUN apk -U upgrade && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*

COPY --from=build /s6 /
# copy s6files and set default to expose all container env to the target app
COPY s6files /etc/
COPY --from=build /dsiem /dsiem

# configs-dist will be used to prepopulate /dsiem/configs if it's mounted off a new empty volume
RUN cp -r /dsiem/configs /dsiem/configs-dist && \
    rm -rf /dsiem/configs/*

ENV S6_KEEP_ENV 1
# fail container if init scripts failed
ENV S6_BEHAVIOUR_IF_STAGE2_FAILS 2

VOLUME ["/dsiem/logs", "/dsiem/configs" ]
EXPOSE 8080
ENTRYPOINT [ "/init"]
