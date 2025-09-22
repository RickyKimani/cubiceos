package main

import (
	cubic "github.com/rickykimani/cubiceos"
	"log"
)

func main() {
	eq := cubic.NewRKCfg(350, 9.4573, 425.1, 37.96, 83.14)
	res, err := cubic.CubicEOS(eq)
	if err != nil {
		log.Fatal(err)
	}
	cubic.ResultPrinter(res)
}
