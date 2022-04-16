FROM golang:1.14-alpine
ENV GOARCH=amd64
RUN apk update && apk add make bash zip
ADD . /build-root
WORKDIR /build-root
CMD make GOARCH=$GOARCH
