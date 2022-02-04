FROM golang:1.16-alpine
ENV GOARCH=amd64
ENV GO111MODULE=off
RUN apk update && apk add make bash zip
ADD . /build-root
WORKDIR /build-root
CMD make GOARCH=$GOARCH
