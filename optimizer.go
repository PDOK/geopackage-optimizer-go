package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
)

func main() {
	log.Println("Starting...")
	sourceGeopackage := flag.String("s", "empty", "source geopackage")

	flag.Parse()
	optimizeGeopackage(*sourceGeopackage)
}

func optimizeGeopackage(sourceGeopackage string) {
	log.Printf("Performing for geopackage: '%s'...\n", sourceGeopackage)
	db := openDb(sourceGeopackage)
	defer db.Close()

	tableNames := getTableNames(db)

	for _, tablename := range tableNames {
		createPuuid(tablename, db)
		createFuuid(tablename, db)
	}
}

func createPuuid(tableName string, db *sql.DB) {
	columnName := "puuid"
	value := "uuid4()"
	addColumn(tableName, columnName, db)
	setColumnValue(tableName, columnName, value, db)
	createIndex(tableName, columnName, db)
}

func createFuuid(tableName string, db *sql.DB) {
	columnName := "fuuid"
	value := fmt.Sprintf("'%s.' || puuid", tableName)
	addColumn(tableName, columnName, db)
	setColumnValue(tableName, columnName, value, db)
	createIndex(tableName, columnName, db)
}
