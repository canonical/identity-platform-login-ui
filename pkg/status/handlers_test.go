package status

import (
	"context"
	"encoding/json"

	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_tracing.go go.opentelemetry.io/otel/trace Tracer
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_status.go -source=./interfaces.go

func TestAliveOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/status", nil)
	w := httptest.NewRecorder()

	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).Times(1).Return(context.TODO(), trace.SpanFromContext(req.Context()))

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockMonitor, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	receivedStatus := new(Status)
	if err := json.Unmarshal(data, receivedStatus); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	if receivedStatus.Status != "ok" {
		t.Fatalf("expected status to be %s not  %s", "ok", receivedStatus.Status)
	}
}

func TestDeepCheckSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/deepcheck", nil)
	w := httptest.NewRecorder()

	mockService.EXPECT().CheckKratosReady(gomock.Any()).Times(1).Return(true, nil)
	mockService.EXPECT().CheckHydraReady(gomock.Any()).Times(1).Return(true, nil)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockMonitor, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	receivedStatus := new(DeepCheckStatus)
	if err := json.Unmarshal(data, receivedStatus); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	if !receivedStatus.KratosStatus {
		t.Fatalf("expected KratosStatus to be %v not %v", true, receivedStatus.KratosStatus)
	}
	if !receivedStatus.HydraStatus {
		t.Fatalf("expected HydraStatus to be %v not %v", true, receivedStatus.HydraStatus)
	}
}

func TestDeepCheckFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/deepcheck", nil)
	w := httptest.NewRecorder()

	mockService.EXPECT().CheckKratosReady(gomock.Any()).Times(1).Return(false, nil)
	mockService.EXPECT().CheckHydraReady(gomock.Any()).Times(1).Return(false, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockMonitor, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	receivedStatus := new(DeepCheckStatus)
	if err := json.Unmarshal(data, receivedStatus); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if receivedStatus.KratosStatus {
		t.Fatalf("expected KratosStatus to be %v not %v", false, receivedStatus.KratosStatus)
	}
	if receivedStatus.HydraStatus {
		t.Fatalf("expected HydraStatus to be %v not %v", false, receivedStatus.HydraStatus)
	}
}
