package main

import (
	"fmt"

	_ "github.com/rancher/prometheus-auth/pkg/data"
	_ "github.com/rancher/prometheus-auth/pkg/prom"
)

var (
	VER  = "dev"
	HASH = "-"
)

func main() {
	fmt.Println("empty main function, just for init pkg/data and pkg/prom")
}
