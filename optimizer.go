package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/creasty/defaults"
)

const (
	pdokNamespace = "098c4e26-6e36-5693-bae9-df35db0bee49"
)

func main() {
	log.Println("Starting...")
	sourceGeopackage := flag.String("s", "empty", "source geopackage")
	serviceType := flag.String("service-type", "ows", "service type to optimize geopackage for")
	config := flag.String("config", "", "optional JSON config for additional optimizations")

	flag.Parse()

	switch *serviceType {
	case "ows":
		optimizeOWSGeopackage(*sourceGeopackage)
	case "oaf":
		optimizeOAFGeopackage(*sourceGeopackage, *config)
	default:
		log.Fatalf("invalid value for service-type: '%s'", *serviceType)
	}
}

func optimizeOAFGeopackage(sourceGeopackage string, config string) {
	log.Printf("Performing OAF optimizations for geopackage: '%s'...\n", sourceGeopackage)
	db := openDb(sourceGeopackage)
	defer db.Close()

	tableNames := getTableNames(db)

	if config != "" {
		var oafConfig OafConfig
		err := json.Unmarshal([]byte(config), &oafConfig)
		if err != nil {
			log.Fatalf("cannot unmarshal oaf config: %s", err)
		}
		err = defaults.Set(&oafConfig)
		if err != nil {
			log.Fatalf("failed to set default config: %s", err)
		}
		for _, tableName := range tableNames {
			if _, ok := oafConfig.Layers[tableName]; !ok {
				log.Printf("WARNING: no config found for gpkg table '%s'", tableName)
				continue
			}
			layerCfg := oafConfig.Layers[tableName]

			// any configured SQL statements are executed first, to allow maximum configuration freedom if needed
			for _, stmt := range layerCfg.SQLStatements {
				executeQuery(stmt, db)
			}

			if layerCfg.ExternalFidColumns != nil {
				addColumn(tableName, "external_fid", "TEXT", db)
				setColumnValue(tableName, "external_fid", fmt.Sprintf("uuid5('%s', '%s'||%s)", pdokNamespace, tableName, strings.Join(layerCfg.ExternalFidColumns, "||")), db)
				createIndex(tableName, []string{"external_fid"}, fmt.Sprintf("%s_external_fid_idx", tableName), false, db)
			}

			if layerCfg.TemporalColumns != nil {
				createIndex(tableName, layerCfg.TemporalColumns, fmt.Sprintf("%s_temporal_idx", tableName), false, db)
			}

			addOAFDefaultOptimizations(tableName, layerCfg.FidColumn, layerCfg.GeomColumn, layerCfg.TemporalColumns, db)

			analyze(db)
		}
	} else {
		for _, tableName := range tableNames {
			addOAFDefaultOptimizations(tableName, "fid", "geom", nil, db)

			analyze(db)
		}
	}
}

func addOAFDefaultOptimizations(tableName string, fidColumn string, geomColumn string, temporalColumns []string, db *sql.DB) {
	addColumn(tableName, "minx", "numeric", db)
	addColumn(tableName, "maxx", "numeric", db)
	addColumn(tableName, "miny", "numeric", db)
	addColumn(tableName, "maxy", "numeric", db)
	setColumnValue(tableName, "minx", fmt.Sprintf("ST_MinX(%s)", geomColumn), db)
	setColumnValue(tableName, "maxx", fmt.Sprintf("ST_MaxX(%s)", geomColumn), db)
	setColumnValue(tableName, "miny", fmt.Sprintf("ST_MinY(%s)", geomColumn), db)
	setColumnValue(tableName, "maxy", fmt.Sprintf("ST_MaxY(%s)", geomColumn), db)

	spatialColumns := []string{fidColumn, "minx", "maxx", "miny", "maxy"}
	if temporalColumns != nil {
		spatialColumns = append(spatialColumns, temporalColumns...)
	}
	createIndex(tableName, spatialColumns, fmt.Sprintf("%s_spatial_idx", tableName), false, db)
}

func optimizeOWSGeopackage(sourceGeopackage string) {
	log.Printf("Performing OWS optimizations for geopackage: '%s'...\n", sourceGeopackage)
	db := openDb(sourceGeopackage)
	defer db.Close()

	tableNames := getTableNames(db)

	for _, tableName := range tableNames {
		columnName := "puuid"
		value := "uuid4()"
		addColumn(tableName, columnName, "TEXT", db)
		setColumnValue(tableName, columnName, value, db)
		createIndex(tableName, []string{columnName}, "", true, db)

		columnName = "fuuid"
		value = fmt.Sprintf("'%s.' || puuid", tableName)
		addColumn(tableName, columnName, "TEXT", db)
		setColumnValue(tableName, columnName, value, db)
		createIndex(tableName, []string{columnName}, "", true, db)
	}
}
