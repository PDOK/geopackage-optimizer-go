FROM ubuntu:20.04 AS build-ext
ENV TZ=Europe/Amsterdam

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

# build uuid extension
RUN apt-get install -y git
RUN git clone https://github.com/PDOK/sqlite3-uuid.git
WORKDIR /sqlite3-uuid
RUN make

FROM golang:1.23-alpine AS build-env

RUN apk update && apk upgrade && \
   apk add --no-cache bash git pkgconfig gcc g++ libc-dev ca-certificates gdal libspatialite sqlite jq libuuid

ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org

ENV TZ=Europe/Amsterdam

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
# hack to make the now 'ancient' sqlite3-uuid lib work
RUN ln -s /usr/lib/libcrypto.so.3 /usr/lib/libcrypto.so.1.1
RUN cp /usr/lib/mod_spatialite.so.8 /usr/lib/mod_spatialite.so

# run tests
RUN go test ./... -covermode=atomic
RUN rm -r testdata/

RUN go build -v -ldflags='-s -w -linkmode auto' -a -installsuffix cgo -o /optimizer .

ENTRYPOINT ["/optimizer", "-s"]
