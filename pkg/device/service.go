package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

type Service struct {
	hydraAdminUrl string

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

type DeviceCodeRequest struct {
	Code string `json:"user_code,omitempty"`
}

type DeviceCodeResponse struct {
	RedirectTo string `json:"redirect_to,omitempty"`
}

func (s *Service) AcceptUserCode(deviceChallenge string, code *DeviceCodeRequest) (*DeviceCodeResponse, error) {
	body, err := json.Marshal(code)
	if err != nil {
		s.logger.Errorf("error when parsing request body: %s", err)
		return nil, err
	}

	url := s.hydraAdminUrl + "/admin/oauth2/auth/requests/device/accept?challenge=" + deviceChallenge
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		s.logger.Errorf("error when constructing request: %s", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Errorf("failed to verify device code: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	acceptDeviceResponse := new(DeviceCodeResponse)
	err = json.Unmarshal(respBody, acceptDeviceResponse)
	if err != nil {
		s.logger.Errorf("error when parsing request body: %s", err)
		return nil, err
	}

	return acceptDeviceResponse, nil
}

func (s *Service) ParseUserCodeBody(r *http.Request) (*DeviceCodeRequest, error) {
	body := new(DeviceCodeRequest)

	err := parseBody(r.Body, &body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func parseBody(b io.ReadCloser, body interface{}) error {
	decoder := json.NewDecoder(b)
	err := decoder.Decode(body)
	return err
}

func NewService(hydraAdminUrl string, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.hydraAdminUrl = hydraAdminUrl

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	return s
}
