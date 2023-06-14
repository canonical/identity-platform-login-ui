package oidc

var OIDCScopeMapping = map[string][]string{
	"openid": {"sub"},
	"profile": {
		"name",
		"family_name",
		"given_name",
		"middle_name",
		"nickname",
		"preferred_username",
		"profile",
		"picture",
		"website",
		"gender",
		"birthdate",
		"zoneinfo",
		"locale",
		"updated_at",
	},
	"email":   {"email", "email_verified"},
	"address": {"address"},
	"phone":   {"phone_number", "phone_number_verified"},
}
