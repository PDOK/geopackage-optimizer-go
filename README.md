# geopackage-optimizer-go

![GitHub license](https://img.shields.io/github/license/PDOK/geopackage-optimizer-go)
[![GitHub release](https://img.shields.io/github/release/PDOK/geopackage-optimizer-go.svg)](https://github.com/PDOK/geopackage-optimizer-go/releases)
[![Go Report Card](https://goreportcard.com/badge/PDOK/geopackage-optimizer-go)](https://goreportcard.com/report/PDOK/geopackage-optimizer-go)
[![Docker Pulls](https://img.shields.io/docker/pulls/pdok/geopackage-optimizer-go.svg)](https://hub.docker.com/r/pdok/geopackage-optimizer-go)

Optimizes geopackage so that it can be used as datasource for (PDOK) OGC services and APIs.

## Build

```
docker build pdok/geopackage-optimizer-go .
```

## Run

```
Usage of /optimizer:
  -config string
        optional JSON config for additional optimizations
  -s string
        source geopackage (default "empty")
  -service-type string
        service type to optimize geopackage for (default "ows")
```

### TL;DR

Run from the root of this repo (note modifies `geopackage/original.gpkg`):

```bash
docker run \
  -v geopackage:/geopackage \
  pdok/geopackage-optimizer-go:latest "/geopackage/original.gpkg"
```

## Optimizations

### OGC webservices

With flag `-service-type ows`:

* create index PUUID using UUID4
* create index FUUID using [tablename].[PUUID]

This ensures that there are randomly generated UUID's usable as index, which has
 a couple of advantages:

* having an index is good for performance
* having a UUID instead of an incremental ID prevents crawling
* having a UUID prevents users from creating applications that assumes that id
  has meaning and will not change in the future

### OGC API Features

With flag `-service-type oaf`:

* create BTree equivalent of an RTree spatial index
* create index for temporal columns
* create indexed column with an "external feature id" (external_fid). This external FID is a UUID v5 based on one or more given columns that are functionally unique across time.

Above optimizations primarily target OGC API Features served through [GoKoala](https://github.com/PDOK/gokoala).