package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed versions.json
var versionsJSON []byte

var Versions map[string]string

func init() {
	if err := json.Unmarshal(versionsJSON, &Versions); err != nil {
		panic(err)
	}
}
