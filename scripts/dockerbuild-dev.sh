#!/bin/bash

builddir=./test/docker-dev
s6file=s6-overlay-amd64.tar.gz
mkdir -p $builddir

([ ! -z "$SKIP_CMD" ] && echo not building cmd || ./scripts/gobuild-cmd.sh ./cmd/dsiem) && \
([ ! -z "$SKIP_WEB" ] && echo not building web || ./scripts/ngbuild-web.sh) && \
cp ./dsiem $builddir/ && \
cp -r ./web/dist $builddir/ && \
cp -r ./deployments/docker/build/s6files $builddir/ && \
cp -r ./configs $builddir/ && \

([ -e "$builddir/$s6file" ] && echo $s6file already exist || wget https://github.com/just-containers/s6-overlay/releases/download/v1.20.0.0/s6-overlay-amd64.tar.gz -O $builddir/$s6file) && \
cd $builddir/ && \
touch Dockerfile && \
{
cat << EOF > Dockerfile
FROM alpine:edge
# Install packages
RUN apk -U upgrade && \ 
    apk add bash ca-certificates wget unzip && \
    rm -rf /var/cache/apk/*

RUN mkdir -p /dsiem/web /dsiem/logs
COPY dsiem /dsiem/dsiem
COPY configs /dsiem/configs
COPY dist /dsiem/web/dist

# configs-dist will be used to prepopulate /dsiem/configs if it's mounted off a new empty volume
RUN cp -r /dsiem/configs /dsiem/configs-dist && rm -rf /dsiem/configs/*

# s6-overlay
COPY s6-overlay-amd64.tar.gz /tmp/
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

EOF
} && docker build . -t defenxor/dsiem-devel
