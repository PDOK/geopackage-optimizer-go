package main

type OwsConfig struct {
	Indices []ManualIndex `json:"indices"`
}

type ManualIndex struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Unique  bool     `json:"unique"`
	Columns []string `json:"columns"`
}
