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
	updateThumb := flag.Bool("update-thumb", false, "update thumbnail")
	flag.Parse()
	// ...
	if *update {
		cmd.UpdateMeta()
	} else if *updateThumb {
		cmd.UpdateThumb()
	} else {
		cmd.RunService()
	}
}
