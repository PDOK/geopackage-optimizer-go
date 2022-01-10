FROM ubuntu:20.04 as build-ext
ENV TZ Europe/Amsterdam
ENV MC_VERSION="RELEASE.2022-01-07T06-01-38Z"

RUN apt-get update && \
    apt-get install -y \
        curl \
        zip \
        gcc \
        make \
        tclsh \
        file \
        libsqlite3-dev \
        libssl-dev \
        uuid-dev

# download minio client
RUN curl https://dl.minio.io/client/mc/release/linux-amd64/archive/mc.${MC_VERSION} > /usr/local/bin/mc && \
    chmod +x /usr/local/bin/mc

# build uuid extension
RUN apt-get install -y git
RUN git clone https://github.com/benwebber/sqlite3-uuid.git
WORKDIR /sqlite3-uuid
RUN make

FROM golang:1.17-alpine AS build-env

RUN apk update && apk upgrade && \
   apk add --no-cache bash git pkgconfig gcc g++ libc-dev ca-certificates gdal libspatialite sqlite jq

ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org

ENV TZ Europe/Amsterdam

WORKDIR /go/src/app

ADD . /go/src/app

# Because of how the layer caching system works in Docker, the go mod download
# command will _ only_ be re-run when the go.mod or go.sum file change
# (or when we add another docker instruction this line)
RUN go mod download
# set crosscompiling fla 0/1 => disabled/enabled
ENV CGO_ENABLED=1
# compile linux only
ENV GOOS=linux

COPY --from=build-ext /sqlite3-uuid/dist/uuid.so.* /usr/lib/uuid.so
COPY --from=build-ext /usr/local/bin/mc /usr/local/bin/mc
RUN cp /usr/lib/mod_spatialite.so.7 /usr/lib/mod_spatialite.so

# run tests
RUN go test ./... -covermode=atomic
RUN rm -r geopackage/

RUN go build -v -ldflags='-s -w -linkmode auto' -a -installsuffix cgo -o /optimizer .

CMD ["/optimizer"]