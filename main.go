/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/amirdaaee/TGMon/cmd"
)

//go:generate swag init --parseDependency --propertyStrategy pascalcase

// @title           TGMon API
// @version         1.0

// @securitydefinitions.apikey ApiKeyAuth
// @in							header
// @name						Authorization
func main() {
	cmd.Execute()
}
