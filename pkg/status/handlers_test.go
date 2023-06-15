package status

import (
	"encoding/json"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	HANDLE_ALIVE_URL = "/health/alive"
)

func TestAliveOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, HANDLE_ALIVE_URL, nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	NewAPI().RegisterEndpoints(mux)

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
