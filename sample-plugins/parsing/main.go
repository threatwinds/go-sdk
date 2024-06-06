package main

import (
	"fmt"
)

type plugin struct{}

func (c *plugin) Run(log *string, cfg *map[string]map[string]interface{}) {
	cn := *cfg
	
	texto := cn["parsing"]["example"].(string)

	fmt.Printf("%s: %s", texto, *log)
}

var Plugin plugin