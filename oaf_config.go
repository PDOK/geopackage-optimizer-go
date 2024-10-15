package main

type OafConfig struct {
	Layers map[string]Layer `json:"layers"`
}

type Layer struct {
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
}
