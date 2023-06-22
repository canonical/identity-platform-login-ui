package config

// EnvSpec is the basic environment configuration setup needed for the app to start
type EnvSpec struct {
	LogLevel string `envconfig:"log_level" default:"error"`
	LogFile  string `envconfig:"log_file" default:"log.txt"`

	KratosPublicURL string `envconfig:"kratos_public_url"`
	HydraAdminURL   string `envconfig:"hydra_admin_url"`
	Port            int    `envconfig:"port" default:"8080"`
}
