package main

import (
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"log"
)

func openDb(sourceGeopackage string) *sql.DB {
	sql.Register("sqlite3_with_extensions", &sqlite3.SQLiteDriver{
		Extensions: []string{
			"mod_spatialite",
			"uuid",
		},
	})

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)

	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}
	return db
}

func getTableNames(db *sql.DB) []string {
	rows, err := db.Query("select table_name from gpkg_contents")

	if err != nil {
		log.Fatalf("error selecting gpkg_contents: %s", err)
	}

	var tableNames []string

	for rows.Next() {
		var table_name string
		err = rows.Scan(&table_name)
		if err != nil {
			log.Fatal(err)
		}
		tableNames = append(tableNames, table_name)
	}
	return tableNames
}

func createIndex(tableName string, columnName string, db *sql.DB) {
	indexName := fmt.Sprintf("%s_%s_index", tableName, columnName)

	query := `CREATE UNIQUE INDEX %v ON %v(%v);`
	fullQuery := fmt.Sprintf(query, indexName, tableName, columnName)
	log.Printf("executing query: %s\n", fullQuery)
	_, err := db.Exec(fullQuery)

	if err != nil {
		log.Fatalf("error creating index: %s", err)
	}
}

func setColumnValue(tableName string, columnName string, value string, db *sql.DB) {
	query := `UPDATE '%v' SET '%v' = %v;`
	fullQuery := fmt.Sprintf(query, tableName, columnName, value)

	log.Printf("executing query: %s\n", fullQuery)

	_, err := db.Exec(fmt.Sprintf(query, tableName, columnName, value))

	if err != nil {
		log.Fatalf("error setting value '%s' to column '%s': '%s'", value, columnName, err)
	}
}

func addColumn(tableName string, columnName string, db *sql.DB) {
	query := "ALTER TABLE '%v' ADD '%v' TEXT;"

	fullQuery := fmt.Sprintf(query, tableName, columnName)

	log.Printf("executing query: %s\n", fullQuery)

	_, err := db.Exec(fullQuery)

	if err != nil {
		log.Fatalf("error adding column '%s': '%s'", columnName, err)
	}
}
