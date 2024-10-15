package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestOptimizeOWSGeopackage(t *testing.T) {
	sourceGeopackage := "geopackage/geopackage.gpkg"
	source, err := os.Open("geopackage/original.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}
	optimizeOWSGeopackage(sourceGeopackage)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	tableNames := getTableNames(db)

	for _, tableName := range tableNames {
		query := "select puuid, fuuid from '%v'"

		fullQuery := fmt.Sprintf(query, tableName)

		rows, err := db.Query(fullQuery)

		if err != nil {
			log.Fatalf("error opening source GeoPackage: %s", err)
		}

		for rows.Next() {
			var puuid string
			var fuuid string
			err = rows.Scan(&puuid, &fuuid)
			if err != nil {
				log.Fatal(err)
			}
			_, err := uuid.Parse(puuid)
			if err != nil {
				log.Fatalf("Generated uuid is invalid because: '%s'", err)
			}
			if fuuid != fmt.Sprintf("%s.%s", tableName, puuid) {
				log.Fatalf("Generated fuuid is invalid because it doesnt match pattern 'tableName.puuid': '%s'", fuuid)
			}
		}
	}
}
