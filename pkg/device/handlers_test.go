package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
	"github.com/go-chi/chi/v5"
	hClient "github.com/ory/hydra-client-go/v2"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_device.go -source=./interfaces.go

func TestHandleDeviceUserCodeAcceptSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	accept := hClient.NewOAuth2RedirectTo("test")

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/device", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(accept, nil)

	mux := chi.NewMux()
	NewAPI(mockService, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	data, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	redirect := hClient.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, redirect); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if redirect.RedirectTo != accept.RedirectTo {
		t.Fatalf("expected %s, got %s.", accept.RedirectTo, redirect.RedirectTo)
	}
}

func TestHandleDeviceUserCodeAcceptParseUserCodeBodyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/device", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleDeviceUserCodeAcceptAcceptUserCodeFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/device", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleDeviceUserCodeInvalidCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/device", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	err := fmt.Errorf("404 Not Found")
	expectedErrorBody := NOT_FOUND_ERROR_DESC + "\n"

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, err)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected HTTP status code 400 got %v", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	errorBody := string(data)
	if errorBody != expectedErrorBody {
		t.Fatalf("expected '%v' got '%v'", expectedErrorBody, string(data))
	}
}

func TestHandleDeviceUserCodeUnexpectedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/device", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 400 got %v", res.StatusCode)
	}
}
