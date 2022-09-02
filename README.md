# geopackage-optimizer-go

![GitHub license](https://img.shields.io/github/license/PDOK/geopackage-optimizer-go)
[![GitHub release](https://img.shields.io/github/release/PDOK/geopackage-optimizer-go.svg)](https://github.com/PDOK/geopackage-optimizer-go/releases)
[![Go Report Card](https://goreportcard.com/badge/PDOK/geopackage-optimizer-go)](https://goreportcard.com/report/PDOK/geopackage-optimizer-go)
[![Docker Pulls](https://img.shields.io/docker/pulls/pdok/geopackage-optimizer-go.svg)](https://hub.docker.com/r/pdok/geopackage-optimizer-go)

Optimizes geopackage so that it can be used as datasource for PDOK ogc services.

## Optimizations

* create index PUUID using UUID4
* create index FUUID using [tablename].[PUUID]

This ensures that there are randomly generated UUID's usable as index, which has
 a couple of advantages:

* having an index is good for performance
* having a UUID instead of an incremental ID prevents crawling
* having a UUID prevents users from creating applications that assumes that id
  has meaning and will not change in the future

## TLDR Usage

Run from the root of this repo (note modifies `geopackage/original.gpkg`):

```bash
gpkg_path=geopackage/original.gpkg
docker run \
  -v "$(realpath $gpkg_path | xargs dirname)":/geopackages \
  -t pdok/geopackage-optimizer-go:latest /optimizer \
  -s "/geopackages/$(basename $gpkg_path)"
```

## Workflow examples

```yaml
spec:
  templates:
    - name: optimize-gpkg
      retryStrategy:
        limit: 2
        retryPolicy: "Always"
        backoff:
          duration: "10"
          factor: 3
      volumes:
        - name: gpkg-volume
          emptyDir: {}
        - name: optimize-gpkg
          configMap:
            name: optimize-gpkg
            defaultMode: 0777
      inputs:
        parameters:
          - name: source-key
          - name: destination-key
      container:
        image: geopackage-optimizer-go
        imagePullPolicy: IfNotPresent
        envFrom:
          - configMapRef:
              name: minio
          - secretRef:
              name: minio
        volumeMounts:
          - name: gpkg-volume
            mountPath: /srv/data
        command: ["/bin/bash", "-c"]
        args:
          - |
            mc alias set minio ${S3_ENDPOINT} ${S3_ACCESS_KEY} ${S3_SECRET_KEY}
            mc cp minio/$(S3_DELIVERY_BUCKET)/{{inputs.parameters.source-key}} /srv/data/transform.gpkg
            /optimizer -s /srv/data/transform.gpkg
            mc cp /srv/data/transform.gpkg minio/$(S3_GEOPACKAGES_BUCKET)/{{inputs.parameters.destination-key}}
        resources:
          limits:
            cpu: "0.1"
            memory: "650Mi"
            ephemeral-storage: #PATCH
          requests:
            cpu: "0.1"
            memory: "650Mi"
            ephemeral-storage: #PATCH
```
