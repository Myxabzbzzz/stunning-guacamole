package main

import (
	"dodobackend/internal/app/apiserver"
	"flag"
	"github.com/BurntSushi/toml"
	"log"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", "/Users/myxabzbzzz/GolandProjects/billing_API/cmd/configs/apiserver.toml", "config path")
}
func main() {
	flag.Parse()

	config := apiserver.NewConfig()
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		log.Fatal(err)
	}

	apiServer := apiserver.NewAPIServer(config)
	if err := apiServer.Run(); err != nil {
		log.Fatal(err)
	}
}
