package authorization

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	fga "github.com/openfga/go-sdk"
)

var ErrInvalidAuthModel = fmt.Errorf("Invalid authorization model schema")

type Authorizer struct {
	Client AuthzClientInterface

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (a *Authorizer) Check(ctx context.Context, user string, relation string, object string) (bool, error) {
	ctx, span := a.tracer.Start(ctx, "authorization.Authorizer.Check")
	defer span.End()

	return a.Client.Check(ctx, user, relation, object)
}

func (a *Authorizer) ListObjects(ctx context.Context, user string, relation string, objectType string) ([]string, error) {
	ctx, span := a.tracer.Start(ctx, "authorization.Authorizer.ListObjects")
	defer span.End()

	return a.Client.ListObjects(ctx, user, relation, objectType)
}

func (a *Authorizer) FilterObjects(ctx context.Context, user string, relation string, objectType string, objs []string) ([]string, error) {
	ctx, span := a.tracer.Start(ctx, "authorization.Authorizer.FilterObjects")
	defer span.End()

	allowedObjs, err := a.ListObjects(ctx, user, relation, objectType)
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, obj := range allowedObjs {
		if contains(objs, obj) {
			ret = append(ret, obj)
		}
	}
	return ret, nil
}

func (a *Authorizer) CreateModel(ctx context.Context) (string, error) {
	ctx, span := a.tracer.Start(ctx, "authorization.Authorizer.CreateModel")
	defer span.End()

	modelId, err := a.Client.WriteModel(ctx, []byte(authModel))
	return modelId, err
}

func (a *Authorizer) ValidateModel(ctx context.Context) error {
	ctx, span := a.tracer.Start(ctx, "authorization.Authorizer.ValidateModel")
	defer span.End()

	var builtinAuthorizationModel fga.AuthorizationModel
	err := json.Unmarshal([]byte(authModel), &builtinAuthorizationModel)
	if err != nil {
		return err
	}

	eq, err := a.Client.CompareModel(ctx, builtinAuthorizationModel)
	if err != nil {
		return err
	}
	if !eq {
		return ErrInvalidAuthModel
	}
	return nil
}

func NewAuthorizer(client AuthzClientInterface, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Authorizer {
	authorizer := new(Authorizer)
	authorizer.Client = client
	authorizer.tracer = tracer
	authorizer.monitor = monitor
	authorizer.logger = logger

	return authorizer
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
