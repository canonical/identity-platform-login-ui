package authorization

import (
	"context"

	fga "github.com/openfga/go-sdk"
)

type AuthorizerInterface interface {
	ListObjects(context.Context, string, string, string) ([]string, error)
	Check(context.Context, string, string, string) (bool, error)
	FilterObjects(context.Context, string, string, string, []string) ([]string, error)
	ValidateModel(context.Context) error
}

type AuthzClientInterface interface {
	ListObjects(context.Context, string, string, string) ([]string, error)
	Check(context.Context, string, string, string) (bool, error)
	ReadModel(context.Context) (*fga.AuthorizationModel, error)
	CompareModel(context.Context, fga.AuthorizationModel) (bool, error)
}
