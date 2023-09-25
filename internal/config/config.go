package config

import (
	"log"

	configuration "github.com/zendesk/zendesk_config_go"
)

type Data struct {
	ProxyPort     int    `default:"5000"`
	AllowHTTP     bool   `default:"false"`
	StatsDHost    string `env:"STATSD_HOST" default:"169.254.1.1"`
	StatsDPort    int    `env:"STATSD_PORT" default:"8125"`
	MetricsPrefix string `default:"istio_ecr_proxy"`
}

func LoadConfig(env []string) (*Data, error) {
	c := &Data{}

	err := configuration.Load(c)
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	return c, nil
}
