package apiserver

type Config struct {
	BindAddr string `toml:"bind_addr"`
	logLevel string `toml:"log_level"`
}

func NewConfig() *Config {
	return &Config{
		BindAddr: ":8080",
		logLevel: "debug",
	}
}
