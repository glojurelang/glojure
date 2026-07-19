package main

import (
	"log"

	"github.com/glojurelang/glojure/internal/deps"
)

func main() {
	dps, err := deps.Load()
	if err != nil {
		log.Fatal(err)
	}
	if dps == nil {
		return
	}
	if err := dps.Gen(); err != nil {
		log.Fatal(err)
	}
}
