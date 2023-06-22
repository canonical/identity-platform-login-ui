package status

import (
	"context"
	"encoding/json"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_tracer.go 	go.opentelemetry.io/otel/trace Tracer

func TestAliveOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/status", nil)
	w := httptest.NewRecorder()

	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).Times(1).Return(context.TODO(), trace.SpanFromContext(req.Context()))

	mux := chi.NewMux()
	NewAPI(mockTracer, mockMonitor, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	receivedStatus := new(Status)
	if err := json.Unmarshal(data, receivedStatus); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, "ok", receivedStatus.Status, "Expected %s, got %s", "ok", receivedStatus.Status)
}
