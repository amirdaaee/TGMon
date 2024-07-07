package main

import (
	"flag"

	"github.com/amirdaaee/TGMon/cmd"
)

func init() {
	cmd.Setup()
}
func main() {
	update := flag.Bool("update", false, "update metadata")
	flag.Parse()
	// ...
	if *update {
		cmd.UpdateMeta()
	} else {
		cmd.RunService()
	}
}
