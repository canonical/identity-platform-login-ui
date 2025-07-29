package config

import "flag"

// EnvSpec is the basic environment configuration setup needed for the app to start
type EnvSpec struct {
	OtelGRPCEndpoint string `envconfig:"otel_grpc_endpoint"`
	OtelHTTPEndpoint string `envconfig:"otel_http_endpoint"`
	TracingEnabled   bool   `envconfig:"tracing_enabled" default:"true"`

	LogLevel string `envconfig:"log_level" default:"error"`
	Debug    bool   `envconfig:"debug" default:"false"`

	Port    int    `envconfig:"port" default:"8080"`
	BaseURL string `envconfig:"base_url" default:""`

	CookiesEncryptionKey string `envconfig:"cookies_encryption_key" required:"true" validate:"required,min=32,max=32"`
	CookieTTL            int    `envconfig:"cookie_ttl" default:"300"`

	KratosPublicURL string `envconfig:"kratos_public_url"`
	KratosAdminURL  string `envconfig:"kratos_admin_url"`
	HydraAdminURL   string `envconfig:"hydra_admin_url"`

	ApiScheme            string `envconfig:"openfga_api_scheme" default:""`
	ApiHost              string `envconfig:"openfga_api_host"`
	ApiToken             string `envconfig:"openfga_api_token"`
	StoreId              string `envconfig:"openfga_store_id"`
	AuthorizationModelId string `envconfig:"openfga_authorization_model_id" default:""`
	AuthorizationEnabled bool   `envconfig:"authorization_enabled" default:"false"`

	MFAEnabled                    bool `envconfig:"mfa_enabled" default:"true"`
	OIDCWebAuthnSequencingEnabled bool `envconfig:"oidc_webauthn_sequencing_enabled" default:"false"`

	SupportEmail string `envconfig:"support_email" default:""`
}

type Flags struct {
	ShowVersion bool
}

func NewFlags() *Flags {
	f := new(Flags)

	flag.BoolVar(&f.ShowVersion, "version", false, "Show the app version and exit")
	flag.Parse()

	return f
}
