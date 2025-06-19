package main

import "log"

type OafConfig struct {
	Layers map[string]Layer `json:"layers"`
}

func (o OafConfig) getLayer(tableName string) (Layer, bool) {
	if _, ok := o.Layers[tableName]; !ok {
		log.Printf("WARNING: no config found for gpkg table '%s'", tableName)
		return Layer{}, false
	}
	return o.Layers[tableName], true
}

type Layer struct {
	FidColumn          string     `json:"fid-column" default:"fid"`
	GeomColumn         string     `json:"geom-column" default:"geom"`
	SQLStatements      []string   `json:"sql-statements"`
	ExternalFidColumns []string   `json:"external-fid-columns"`
	TemporalColumns    []string   `json:"temporal-columns"`
	Relations          []Relation `json:"relations"`
}

type Relation struct {
	Table   string          `json:"table"`
	Columns RelationColumns `json:"columns"`
}

type RelationColumns struct {
	ForeignKey string `json:"fk"`
	PrimaryKey string `json:"pk"`
	Prefix     string `json:"prefix"`
}

func (r *Relation) ColumnName() string {
	result := r.Table
	if r.Columns.Prefix != "" {
		result += "_" + r.Columns.Prefix
	}
	result += "_external_fid"
	return result
}
