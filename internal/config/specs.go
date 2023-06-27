package config

// EnvSpec is the basic environment configuration setup needed for the app to start
type EnvSpec struct {
	JaegerEndpoint string `envconfig:"jaeger_endpoint"`

	LogLevel string `envconfig:"log_level" default:"error"`
	LogFile  string `envconfig:"log_file" default:"log.txt"`

	Port int `envconfig:"port" default:"8080"`

	KratosPublicURL string `envconfig:"kratos_public_url"`
	HydraAdminURL   string `envconfig:"hydra_admin_url"`
}
