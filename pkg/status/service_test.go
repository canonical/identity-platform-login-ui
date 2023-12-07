package status

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"go.uber.org/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_status.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_tracing.go go.opentelemetry.io/otel/trace Tracer
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_kratos.go -mock_names MetadataApi=MockKratosMetadataApi github.com/ory/kratos-client-go MetadataApi
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_hydra.go -mock_names MetadataApi=MockHydraMetadataApi "github.com/ory/hydra-client-go/v2" MetadataApi

func TestKratosReadySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratos := NewMockKratosMetadataApi(ctrl)
	mockHydra := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	kratosIsReadyReturn := kClient.MetadataApiIsReadyRequest{
		ApiService: mockKratos,
	}

	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(1.0)).Times(1)
	mockTracer.EXPECT().Start(gomock.Any(), "status.Service.kratosReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().IsReady(gomock.Any()).Times(1).Return(kratosIsReadyReturn)
	mockKratos.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.MetadataApiIsReadyRequest) (*kClient.IsAlive200Response, *http.Response, error) {
			isAlive := kClient.NewIsAlive200ResponseWithDefaults()
			httpResp := new(http.Response)
			httpResp.StatusCode = http.StatusOK
			return isAlive, httpResp, nil
		},
	)

	status, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).kratosReady(ctx)

	if !status {
		t.Fatalf("expected status to be %v not  %v", true, status)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestHydraReadySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratos := NewMockKratosMetadataApi(ctrl)
	mockHydra := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	hydraIsReadyReturn := hClient.MetadataApiIsReadyRequest{
		ApiService: mockHydra,
	}

	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(1.0)).Times(1)
	mockTracer.EXPECT().Start(gomock.Any(), "status.Service.hydraReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().IsReady(gomock.Any()).Times(1).Return(hydraIsReadyReturn)
	mockHydra.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.MetadataApiIsReadyRequest) (*hClient.IsReady200Response, *http.Response, error) {
			isReady := hClient.NewIsReady200ResponseWithDefaults()
			httpResp := new(http.Response)
			httpResp.StatusCode = http.StatusOK
			return isReady, httpResp, nil
		},
	)

	status, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).hydraReady(ctx)

	if !status {
		t.Fatalf("expected status to be %v not  %v", true, status)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestKratosReadyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratos := NewMockKratosMetadataApi(ctrl)
	mockHydra := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	kratosIsReadyReturn := kClient.MetadataApiIsReadyRequest{
		ApiService: mockKratos,
	}

	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(0.0)).Times(1)
	mockTracer.EXPECT().Start(gomock.Any(), "status.Service.kratosReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().IsReady(gomock.Any()).Times(1).Return(kratosIsReadyReturn)
	mockKratos.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.MetadataApiIsReadyRequest) (*kClient.IsAlive200Response, *http.Response, error) {
			httpResp := new(http.Response)
			httpResp.StatusCode = http.StatusInternalServerError
			return nil, httpResp, fmt.Errorf("error")
		},
	)

	status, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).kratosReady(ctx)

	if status {
		t.Fatalf("expected status to be %v not  %v", false, status)
	}

	if err == nil {
		t.Fatalf("expected error not to be nil")
	}
}

func TestHydraReadyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratos := NewMockKratosMetadataApi(ctrl)
	mockHydra := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	hydraIsReadyReturn := hClient.MetadataApiIsReadyRequest{
		ApiService: mockHydra,
	}

	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(0.0)).Times(1)
	mockTracer.EXPECT().Start(gomock.Any(), "status.Service.hydraReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().IsReady(gomock.Any()).Times(1).Return(hydraIsReadyReturn)
	mockHydra.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.MetadataApiIsReadyRequest) (*hClient.IsReady200Response, *http.Response, error) {
			httpResp := new(http.Response)
			httpResp.StatusCode = http.StatusInternalServerError
			return nil, httpResp, fmt.Errorf("error")
		},
	)

	status, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).hydraReady(ctx)

	if status {
		t.Fatalf("expected status to be %v not  %v", false, status)
	}

	if err == nil {
		t.Fatalf("expected error not to be nil")
	}
}
