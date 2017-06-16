FROM golang:1.8-alpine
RUN apk update && apk add make bash zip
ADD . /build-root
WORKDIR /build-root
CMD make
