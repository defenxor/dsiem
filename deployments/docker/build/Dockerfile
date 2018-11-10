FROM alpine:edge
ARG ver
ENV DSIEM_VER=$ver

# Install packages
RUN apk -U upgrade && \
    apk add bash ca-certificates wget unzip && \
    rm -rf /var/cache/apk/*

# temporary, get fetch and download from private repo
ARG token
ENV GITHUB_OAUTH_TOKEN=$token
RUN wget https://github.com/gruntwork-io/fetch/releases/download/v0.3.2/fetch_linux_amd64 -O /tmp/fetch && chmod +x /tmp/fetch && \
    /tmp/fetch --repo="https://github.com/defenxor/dsiem" --tag=$DSIEM_VER --release-asset="dsiem-server_linux_amd64.zip" /tmp && \
    unzip /tmp/dsiem-server_linux_amd64.zip -d /dsiem && rm -rf /tmp/dsiem-server_linux_amd64.zip /tmp/fetch

# To be used later, replacing the above
# RUN wget https://github.com/defenxor/dsiem/releases/download/$DSIEM_VER/dsiem-server-linux-amd64.zip -O /tmp/dsiem.zip && \
#    unzip /tmp/dsiem.zip -d /dsiem && rm -rf /tmp/dsiem.zip

# configs-dist will be used to prepolulate /dsiem/configs if it's mounted off a new empty volume
RUN cp -r /dsiem/configs /dsiem/configs-dist

# s6-overlay
ADD https://github.com/just-containers/s6-overlay/releases/download/v1.20.0.0/s6-overlay-amd64.tar.gz /tmp/
RUN tar xzf /tmp/s6-overlay-amd64.tar.gz -C /

ENV TERM xterm-256color
# copy s6files and set default to expose all container env to the target app
ADD s6files /etc/
ENV S6_KEEP_ENV 1
# fail container if init scripts failed
ENV S6_BEHAVIOUR_IF_STAGE2_FAILS 2

VOLUME ["/dsiem/logs", "/dsiem/configs" ]
EXPOSE 8080
ENTRYPOINT [ "/init"]
