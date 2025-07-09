// @title Billing API
// @version 1.0
// @description RESTful API for user accounts and transactions
// @host localhost:8080
// @BasePath /

package main

import (
	"flag"
	"log"

	"billing_API/internal/app/apiserver"

	_ "billing_API/docs"

	"github.com/BurntSushi/toml"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", "/Users/myxabzbzzz/GolandProjects/billing_API/configs/apiserver.toml", "path to config file")
}

func main() {
	flag.Parse()

	config := apiserver.NewConfig()
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		log.Fatal(err)
	}

	if err := apiserver.StartWithSwagger(config); err != nil {
		log.Fatal(err)
	}
}
