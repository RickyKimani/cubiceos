package main

import (
	"log"

	"github.com/rickykimani/cubiceos"
)

func main() {
	eq := cubiceos.NewRKCfg(
		350,    // T (K)
		9.4573, // P (bar)
		425.1,  // Tc (K)
		37.96,  // Pc (bar)
		83.14,  // R (bar•cm^3/(mol•K))
	)
	res, err := cubiceos.CubicEOS(eq)
	if err != nil {
		log.Fatal(err)
	}
	cubiceos.ResultPrinter(res)
}
