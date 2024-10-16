package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/mattn/go-sqlite3"
)

func registerDriver(driverName string, extensions []string) {
	for _, driver := range sql.Drivers() {
		if driver == driverName {
			return
		}
	}
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		Extensions: extensions,
	})
}

func openDb(sourceGeopackage string) *sql.DB {
	registerDriver(
		"sqlite3_with_extensions",
		[]string{
			"mod_spatialite",
			"uuid",
		},
	)

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

func createIndex(tableName string, columnNames []string, indexName string, unique bool, db *sql.DB) {
	if indexName == "" {
		indexName = fmt.Sprintf("%s_%s_index", tableName, strings.Join(columnNames, "_"))
	}

	var queryStr string
	if unique {
		queryStr = "CREATE UNIQUE INDEX %s ON %s(%s);"
	} else {
		queryStr = "CREATE INDEX %s ON %s(%s);"
	}

	query := fmt.Sprintf(queryStr, indexName, tableName, strings.Join(columnNames, ","))
	log.Printf("executing query: %s\n", query)

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("error creating index: %s", err)
	}
}

func setColumnValue(tableName string, columnName string, value string, db *sql.DB) {
	query := fmt.Sprintf("UPDATE '%s' SET '%s' = %s;", tableName, columnName, value)
	log.Printf("executing query: %s\n", query)

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("error setting value '%s' to column '%s': '%s'", value, columnName, err)
	}
}

func addColumn(tableName string, columnName string, columnType string, db *sql.DB) {
	query := fmt.Sprintf("ALTER TABLE '%s' ADD '%s' %s;", tableName, columnName, columnType)
	log.Printf("executing query: %s\n", query)

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("error adding column '%s': '%s'", columnName, err)
	}
}

func executeQuery(query string, db *sql.DB) {
	query = fmt.Sprintf("%s;", query)
	log.Printf("executing query: %s\n", query)

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("error executing query: '%s'", err)
	}
}

func analyze(db *sql.DB) {
	_, err := db.Exec("ANALYZE")
	if err != nil {
		log.Fatalf("error running analyze: %s", err)
	}
}
