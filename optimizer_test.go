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
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_ows.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}

	optimizeOWSGeopackage(sourceGeopackage, "")

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	tableNames := getTableNames(db)

	for _, tableName := range tableNames {
		query := fmt.Sprintf("select puuid, fuuid from '%v'", tableName)

		rows, err := db.Query(query)
		if err != nil {
			log.Fatalf("error executing query: %s", err)
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

func TestOptimizeOAFGeopackageNoConfig(t *testing.T) {
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_oaf.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}

	config := ""
	optimizeOAFGeopackage(sourceGeopackage, config)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	// check spatial columns
	rows, err := db.Query("select minx, maxx, miny, maxy from 'pand';")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var minx, maxx, miny, maxy string
		err = rows.Scan(&minx, &maxx, &miny, &maxy)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
	}

	// check spatial index
	rows, err = db.Query("select exists(select 1 from sqlite_master where type = 'index' and name = 'pand_spatial_idx' and tbl_name = 'pand') as index_exists;")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var exists int
		err = rows.Scan(&exists)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		if exists != 1 {
			log.Fatal("spatial index missing for table 'pand'")
		}
	}
}

func TestOptimizeOAFGeopackageExternalFid(t *testing.T) {
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_oaf.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}
	config := `{
	  "layers":
	  {
	    "pand":
	    {
	      "external-fid-columns":
	      [
	        "identificatie"
	      ]
	    }
	  }
	}`
	optimizeOAFGeopackage(sourceGeopackage, config)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	rows, err := db.Query("select external_fid from 'pand';")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var externalFid string
		err = rows.Scan(&externalFid)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		_, err := uuid.Parse(externalFid)
		if err != nil {
			log.Fatalf("'external_fid' is invalid because: '%s'", err)
		}
	}
}

func TestOptimizeOAFGeopackageSQLStatements(t *testing.T) {
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_oaf.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}

	config := `{
	  "layers":
	  {
	    "pand":
	    {
	      "sql-statements":
	      [
	        "ALTER TABLE pand ADD COLUMN fid_copy integer",
	        "UPDATE pand SET fid_copy = fid",
	        "CREATE INDEX pand_identificatie_idx ON pand(identificatie)"
	      ]
	    }
	  }
	}`
	optimizeOAFGeopackage(sourceGeopackage, config)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	// check copied column
	rows, err := db.Query("select fid, fid_copy from 'pand';")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var fid, fidCopy string
		err = rows.Scan(&fid, &fidCopy)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		if fid != fidCopy {
			log.Fatalf("row invalid: '%s' != '%s'", fid, fidCopy)
		}
	}

	// check specified index
	rows, err = db.Query("select exists(select 1 from sqlite_master where type = 'index' and name = 'pand_identificatie_idx' and tbl_name = 'pand') as index_exists;")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var exists int
		err = rows.Scan(&exists)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		if exists != 1 {
			log.Fatal("index 'pand_identificatie_idx' is missing")
		}
	}
}

func TestOptimizeOAFGeopackageRelations(t *testing.T) {
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_oaf.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}

	config := `{
	  "layers":
	  {
	    "pand":
	    {
	      "external-fid-columns":
	      [
	        "identificatie"
	      ]
	    },
	    "other":
	    {
	      "external-fid-columns":
	      [
	        "fid"
	      ],
          "relations": 
          [
			{
				"table": "pand",
	            "columns": 
                {
                  "keys": [
					{
	                  "fk": "fk",
	                  "pk": "identificatie"
					}
				  ]
	            }
            }
          ]
	    }
	  }
	}`
	optimizeOAFGeopackage(sourceGeopackage, config)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	rows, err := db.Query("select pand_external_fid from other;")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var otherExternalFid string
		err = rows.Scan(&otherExternalFid)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		actual, err := uuid.Parse(otherExternalFid)
		if err != nil {
			log.Fatalf("'pand_external_fid' is invalid because: '%s'", err)
		}
		expected := "a64649db-70c5-518f-a842-26ce86113d52"
		if actual.String() != expected {
			log.Fatalf("expected fk: '%s', got '%s'", expected, actual)
		}
	}
}

func TestOptimizeOAFGeopackageRelationsWithCompositeKey(t *testing.T) {
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_oaf.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}

	config := `{
	  "layers":
	  {
	    "pand":
	    {
	      "external-fid-columns":
	      [
	        "identificatie"
	      ]
	    },
	    "other_composite":
	    {
	      "external-fid-columns":
	      [
	        "fid"
	      ],
          "relations": 
          [
			{
				"table": "pand",
	            "columns": 
                {
                  "keys": [
					{
	                  "fk": "bouwjaar",
	                  "pk": "bouwjaar"
					},
					{
	                  "fk": "status",
	                  "pk": "status"
					}
				  ]
	            }
            }
          ]
	    }
	  }
	}`
	optimizeOAFGeopackage(sourceGeopackage, config)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	rows, err := db.Query("select pand_external_fid from other_composite;")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var otherExternalFid string
		err = rows.Scan(&otherExternalFid)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		actual, err := uuid.Parse(otherExternalFid)
		if err != nil {
			log.Fatalf("'pand_external_fid' is invalid because: '%s'", err)
		}
		expected := "7ef33d14-4114-53e6-aaed-d25961c06c61"
		if actual.String() != expected {
			log.Fatalf("expected fk: '%s', got '%s'", expected, actual)
		}
	}
}

func TestOptimizeOAFGeopackageFullConfig(t *testing.T) {
	sourceGeopackage := "testdata/geopackage.gpkg"
	source, err := os.Open("testdata/original_oaf.gpkg")
	if err != nil {
		log.Fatalf("error opening source GeoPackage: %s", err)
	}

	destination, _ := os.Create(sourceGeopackage)
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatalf("error copying GeoPackage: %s", err)
	}

	config := `{
	  "layers":
	  {
	    "pand":
	    {
	      "fid-column": "feature_id",
	      "geom-column": "geom",
	      "sql-statements":
	      [
	        "ALTER TABLE pand RENAME COLUMN fid TO feature_id",
	        "CREATE INDEX pand_identificatie_idx ON pand(identificatie)"
	      ],
	      "external-fid-columns":
	      [
	        "identificatie"
	      ],
	      "temporal-columns":
	      [
	        "bouwjaar"
	      ]
	    }
	  }
	}`
	optimizeOAFGeopackage(sourceGeopackage, config)

	db, err := sql.Open("sqlite3_with_extensions", sourceGeopackage)
	if err != nil {
		log.Fatalf("error opening sourceGeoPackage: %s", err)
	}
	defer db.Close()

	// check temporal index exists
	rows, err := db.Query("select exists(select 1 from sqlite_master where type = 'index' and name = 'pand_temporal_idx' and tbl_name = 'pand') as index_exists;")
	if err != nil {
		log.Fatalf("error executing query: %s", err)
	}

	for rows.Next() {
		var exists int
		err = rows.Scan(&exists)
		if err != nil {
			log.Fatalf("error scanning row: %s", err)
		}
		if exists != 1 {
			log.Fatal("index 'pand_temporal_idx' is missing")
		}
	}
}
