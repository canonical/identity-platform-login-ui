package openfga

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	openfga "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
)

type Config struct {
	ApiScheme   string
	ApiHost     string
	StoreID     string
	ApiToken    string
	AuthModelID string
	Debug       bool

	Tracer  tracing.TracingInterface
	Monitor monitoring.MonitorInterface
	Logger  logging.LoggerInterface
}

func NewConfig(apiScheme, apiHost, storeID, apiToken, authModelID string, debug bool, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Config {
	c := new(Config)

	c.ApiScheme = apiScheme
	c.ApiHost = apiHost
	c.StoreID = storeID
	c.ApiToken = apiToken
	c.AuthModelID = authModelID
	c.Debug = debug

	c.Monitor = monitor
	c.Tracer = tracer
	c.Logger = logger

	return c
}

type Client struct {
	storeID string

	c *client.OpenFgaClient

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (c *Client) APIClient() *client.OpenFgaClient {
	return c.c
}

func (c *Client) ReadModel(ctx context.Context) (*openfga.AuthorizationModel, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.Client.ReadModel")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	authModel, err := c.c.ReadAuthorizationModelExecute(c.c.ReadAuthorizationModel(ctx))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return authModel.AuthorizationModel, nil
}

func (c *Client) WriteModel(ctx context.Context, model []byte) (string, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.Client.WriteModel")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	authModel := new(client.ClientWriteAuthorizationModelRequest)

	if err := json.Unmarshal(model, authModel); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	data, err := c.c.WriteAuthorizationModelExecute(
		c.c.WriteAuthorizationModel(ctx).Body(*authModel),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	return data.GetAuthorizationModelId(), nil
}

func (c *Client) ListObjects(ctx context.Context, user string, relation string, objectType string) ([]string, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.Client.ListObjects")
	defer span.End()

	r := c.APIClient().OpenFgaApi.ListObjects(ctx, c.storeID)
	body := &openfga.ListObjectsRequest{
		User:     user,
		Relation: relation,
		Type:     objectType,
	}
	r = r.Body(*body)
	objectsResponse, _, err := c.APIClient().OpenFgaApi.ListObjectsExecute(r)
	if err != nil {
		c.logger.Errorf("issues performing list operation: %s", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	allowedObjs := make([]string, len(objectsResponse.GetObjects()))
	for i, p := range objectsResponse.GetObjects() {
		// remove the "{objectType}:" prefix from the response
		allowedObjs[i] = p[len(fmt.Sprintf("%s:", objectType)):]
	}

	span.SetStatus(codes.Ok, "")
	return allowedObjs, nil
}

func (c *Client) Check(ctx context.Context, user string, relation string, object string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.Client.Check")
	defer span.End()

	r := c.APIClient().OpenFgaApi.Check(ctx, c.storeID)
	body := openfga.NewCheckRequest(
		openfga.CheckRequestTupleKey{
			User:     user,
			Relation: relation,
			Object:   object,
		},
	)
	r = r.Body(*body)

	check, _, err := c.APIClient().OpenFgaApi.CheckExecute(r)
	if err != nil {
		c.logger.Errorf("issues performing check operation: %s", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	span.SetStatus(codes.Ok, "")
	return check.GetAllowed(), nil
}

func (c *Client) CompareModel(ctx context.Context, model openfga.AuthorizationModel) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.Client.ReadModel")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	authModel, err := c.ReadModel(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	if authModel.SchemaVersion != model.SchemaVersion {
		c.logger.Errorf("invalid authorization model schema version")
		span.SetStatus(codes.Ok, "")
		return false, nil
	}
	if reflect.DeepEqual(authModel.TypeDefinitions, model.TypeDefinitions) {
		c.logger.Errorf("invalid authorization model type definitions")
		span.SetStatus(codes.Ok, "")
		return false, nil
	}

	span.SetStatus(codes.Ok, "")
	return true, nil
}

func NewClient(cfg *Config) *Client {
	c := new(Client)

	if cfg == nil {
		panic("OpenFGA config missing")
	}

	fga, err := client.NewSdkClient(
		&client.ClientConfiguration{
			ApiScheme: cfg.ApiScheme,
			ApiHost:   cfg.ApiHost,
			StoreId:   cfg.StoreID,
			Credentials: &credentials.Credentials{
				Method: credentials.CredentialsMethodApiToken,
				Config: &credentials.Config{
					ApiToken: cfg.ApiToken,
				},
			},
			AuthorizationModelId: cfg.AuthModelID,
			Debug:                cfg.Debug,
			HTTPClient:           &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("issues setting up OpenFGA client %s", err))
	}

	c.c = fga
	c.storeID = cfg.StoreID
	c.tracer = cfg.Tracer
	c.monitor = cfg.Monitor
	c.logger = cfg.Logger

	return c
}
