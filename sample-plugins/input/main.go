package main

import (
	"fmt"
	"time"
)

type plugin struct{}

func (c *plugin) Run(channel chan []byte, cfg *map[string]map[string]interface{}) {
	for {
		cn := *cfg

		texto := cn["input"]["example"].(string)

		log := fmt.Sprintf(`{
			"dataSource": "28.12.199.2",
			"dataType": "linux",
			"tenantId": "999c898a-cf9a-4ff4-8183-6aa81cc477c1",
			"timestamp": "2024-06-01T11:46:22.417666Z",
			"raw": "%s"
		}`, texto)

		channel <- []byte(log)

		time.Sleep(10 * time.Second)
	}
}

var Plugin plugin
