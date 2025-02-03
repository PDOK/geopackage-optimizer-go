package main

type OwsConfig struct {
	Indices []ManualIndex
}

type ManualIndex struct {
	Name    string
	Table   string
	Unique  bool
	Columns []string
}
