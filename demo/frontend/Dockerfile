FROM node:12-alpine as builder

RUN apk update && apk add git && apk upgrade 

COPY web /web
WORKDIR /web
RUN npm install
RUN npm run build

FROM nginx:alpine

RUN apk update && apk upgrade && \
  apk add --update ca-certificates && \
  rm /var/cache/apk/*

WORKDIR /usr/share/nginx/html
COPY --from=builder /web/build .
