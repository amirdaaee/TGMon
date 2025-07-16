/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/amirdaaee/TGMon/cmd"
)

//go:generate swag init --parseDependency

// @securitydefinitions.apikey ApiKeyAuth
// @in							header
// @name						Authorization
func main() {
	cmd.Execute()
}
