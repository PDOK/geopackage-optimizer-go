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
		optimizeOWSGeopackage(*sourceGeopackage, *config)
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

	tables := readTables(db)

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
		for _, table := range tables {
			layerCfg, ok := oafConfig.getLayer(table.Name)
			if !ok {
				continue
			}

			// any configured SQL statements are executed first, to allow maximum configuration freedom if needed
			for _, stmt := range layerCfg.SQLStatements {
				executeQuery(stmt, db)
			}

			// add external_fid column, then set it to uuid5 based on concatenation of collection name and content of given columns, and create an index on it
			if layerCfg.ExternalFidColumns != nil {
				addColumn(table.Name, "external_fid", "TEXT", db)
				setColumnValue(table.Name, "external_fid", fmt.Sprintf("uuid5('%s', '%s'||%s)", pdokNamespace, table.Name, strings.Join(layerCfg.ExternalFidColumns, "||")), db)
				createIndex(table.Name, []string{"external_fid"}, fmt.Sprintf("%s_external_fid_idx", table.Name), false, db)
			}

			if layerCfg.TemporalColumns != nil {
				createIndex(table.Name, layerCfg.TemporalColumns, fmt.Sprintf("%s_temporal_idx", table.Name), false, db)
			}

			addOAFDefaultOptimizations(table, layerCfg.FidColumn, layerCfg.GeomColumn, layerCfg.TemporalColumns, db)
		}
		addRelations(tables, oafConfig, db)
	} else {
		for _, table := range tables {
			addOAFDefaultOptimizations(table, "fid", "geom", nil, db)
		}
	}

	// finally, optimize db by gathering statistics
	analyze(db)
}

func addOAFDefaultOptimizations(table Table, fidColumn string, geomColumn string, temporalColumns []string, db *sql.DB) {
	if !table.IsFeatures {
		log.Printf("Skipping spatial optimizations for table '%s' because it is not of type 'features'", table.Name)
		return
	}
	addColumn(table.Name, "minx", "numeric", db)
	addColumn(table.Name, "maxx", "numeric", db)
	addColumn(table.Name, "miny", "numeric", db)
	addColumn(table.Name, "maxy", "numeric", db)
	setColumnValue(table.Name, "minx", fmt.Sprintf("ST_MinX(%s)", geomColumn), db)
	setColumnValue(table.Name, "maxx", fmt.Sprintf("ST_MaxX(%s)", geomColumn), db)
	setColumnValue(table.Name, "miny", fmt.Sprintf("ST_MinY(%s)", geomColumn), db)
	setColumnValue(table.Name, "maxy", fmt.Sprintf("ST_MaxY(%s)", geomColumn), db)

	spatialColumns := []string{fidColumn, "minx", "maxx", "miny", "maxy"}
	if temporalColumns != nil {
		spatialColumns = append(spatialColumns, temporalColumns...)
	}
	createIndex(table.Name, spatialColumns, fmt.Sprintf("%s_spatial_idx", table.Name), false, db)
}

func addRelations(tables []Table, oafConfig OafConfig, db *sql.DB) {
	for _, table := range tables {
		layerCfg, ok := oafConfig.getLayer(table.Name)
		if !ok {
			continue
		}

		// now that every table contains an external_fid, add relations when specified.
		if layerCfg.ExternalFidColumns != nil && layerCfg.Relations != nil {
			for _, relation := range layerCfg.Relations {
				log.Printf("Adding relation: %s -> %s.external_fid", relation.ColumnName(), relation.Table)
				addColumn(table.Name, relation.ColumnName(), "TEXT", db)

				if len(relation.Columns.Keys) < 1 {
					log.Fatalf("relation '%s' must have at least one pk/fk defined", relation.ColumnName())
				}
				// build and execute SQL query to fill the newly added column with external feature ID of the referenced table
				whereClause := ""
				for i, key := range relation.Columns.Keys {
					if i > 0 {
						whereClause += " and "
					}
					whereClause += fmt.Sprintf("%s.%s = t.%s", table.Name, key.ForeignKey, key.PrimaryKey)
				}
				executeQuery(fmt.Sprintf("update %s set %s = (select t.external_fid from %s t where %s)",
					table.Name, relation.ColumnName(), relation.Table, whereClause), db)
			}
		}
	}
}

func optimizeOWSGeopackage(sourceGeopackage string, config string) {
	log.Printf("Performing OWS optimizations for geopackage: '%s'...\n", sourceGeopackage)
	db := openDb(sourceGeopackage)
	defer db.Close()

	tables := readTables(db)

	for _, table := range tables {
		columnName := "puuid"
		value := "uuid4()"
		addColumn(table.Name, columnName, "TEXT", db)
		setColumnValue(table.Name, columnName, value, db)
		createIndex(table.Name, []string{columnName}, "", true, db)

		columnName = "fuuid"
		value = fmt.Sprintf("'%s.' || puuid", table.Name)
		addColumn(table.Name, columnName, "TEXT", db)
		setColumnValue(table.Name, columnName, value, db)
		createIndex(table.Name, []string{columnName}, "", true, db)
	}

	if config != "" {
		var owsConfig OwsConfig
		err := json.Unmarshal([]byte(config), &owsConfig)
		if err != nil {
			log.Fatalf("cannot unmarshal ows config: %s", err)
		}
		if len(owsConfig.Indices) > 0 {
			foundNames := make(map[string]bool)
			for _, index := range owsConfig.Indices {
				if foundNames[index.Name] {
					log.Fatalf("Index name '%s' was found more than once", index.Name)
				}
				foundNames[index.Name] = true
			}

			for _, index := range owsConfig.Indices {
				createIndex(index.Table, index.Columns, index.Name, index.Unique, db)
			}
		}
	}
}
