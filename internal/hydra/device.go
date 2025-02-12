package hydra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"
)

// We implement the device API logic, because the upstream sdk does not support it.
// Otherwise we would have to fork the sdk
// TODO(nsklikas): Remove this once upstream hydra supports the device flow

type DeviceApiService struct {
	client *hClient.APIClient
	hClient.OAuth2API
}

type APIError struct {
	body  []byte
	error string
}

// Error returns non-empty string if there was an error.
func (e APIError) Error() string {
	return e.error
}

// Body returns the raw bytes of the response
func (e APIError) Body() []byte {
	return e.body
}

// Prevent trying to import "fmt"
func reportError(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

// AcceptDeviceUserCodeRequest Contains information on an device verification
type AcceptDeviceUserCodeRequest struct {
	UserCode *string `json:"user_code,omitempty"`
}

// NewAcceptDeviceUserCodeRequest instantiates a new AcceptDeviceUserCodeRequest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewAcceptDeviceUserCodeRequest() *AcceptDeviceUserCodeRequest {
	this := AcceptDeviceUserCodeRequest{}
	return &this
}

// NewAcceptDeviceUserCodeRequestWithDefaults instantiates a new AcceptDeviceUserCodeRequest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewAcceptDeviceUserCodeRequestWithDefaults() *AcceptDeviceUserCodeRequest {
	this := AcceptDeviceUserCodeRequest{}
	return &this
}

// GetUserCode returns the UserCode field value if set, zero value otherwise.
func (o *AcceptDeviceUserCodeRequest) GetUserCode() string {
	if o == nil || o.UserCode == nil {
		var ret string
		return ret
	}
	return *o.UserCode
}

// GetUserCodeOk returns a tuple with the UserCode field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AcceptDeviceUserCodeRequest) GetUserCodeOk() (*string, bool) {
	if o == nil || o.UserCode == nil {
		return nil, false
	}
	return o.UserCode, true
}

// HasUserCode returns a boolean if a field has been set.
func (o *AcceptDeviceUserCodeRequest) HasUserCode() bool {
	if o != nil && o.UserCode != nil {
		return true
	}

	return false
}

// SetUserCode gets a reference to the given string and assigns it to the UserCode field.
func (o *AcceptDeviceUserCodeRequest) SetUserCode(v string) {
	o.UserCode = &v
}

func (o AcceptDeviceUserCodeRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UserCode != nil {
		toSerialize["user_code"] = o.UserCode
	}
	return json.Marshal(toSerialize)
}

type NullableAcceptDeviceUserCodeRequest struct {
	value *AcceptDeviceUserCodeRequest
	isSet bool
}

func (v NullableAcceptDeviceUserCodeRequest) Get() *AcceptDeviceUserCodeRequest {
	return v.value
}

func (v *NullableAcceptDeviceUserCodeRequest) Set(val *AcceptDeviceUserCodeRequest) {
	v.value = val
	v.isSet = true
}

func (v NullableAcceptDeviceUserCodeRequest) IsSet() bool {
	return v.isSet
}

func (v *NullableAcceptDeviceUserCodeRequest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAcceptDeviceUserCodeRequest(val *AcceptDeviceUserCodeRequest) *NullableAcceptDeviceUserCodeRequest {
	return &NullableAcceptDeviceUserCodeRequest{value: val, isSet: true}
}

func (v NullableAcceptDeviceUserCodeRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAcceptDeviceUserCodeRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

type ApiAcceptUserCodeRequestRequest struct {
	ctx                         context.Context
	ApiService                  OAuth2API
	deviceChallenge             *string
	acceptDeviceUserCodeRequest *AcceptDeviceUserCodeRequest
}

func (r ApiAcceptUserCodeRequestRequest) DeviceChallenge(deviceChallenge string) ApiAcceptUserCodeRequestRequest {
	r.deviceChallenge = &deviceChallenge
	return r
}

func (r ApiAcceptUserCodeRequestRequest) AcceptDeviceUserCodeRequest(acceptDeviceUserCodeRequest AcceptDeviceUserCodeRequest) ApiAcceptUserCodeRequestRequest {
	r.acceptDeviceUserCodeRequest = &acceptDeviceUserCodeRequest
	return r
}

func (r ApiAcceptUserCodeRequestRequest) Execute() (*hClient.OAuth2RedirectTo, *http.Response, error) {
	return r.ApiService.AcceptUserCodeRequestExecute(r)
}

/*
AcceptUserCodeRequest Accepts a device grant user_code request

Accepts a device grant user_code request

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiAcceptUserCodeRequestRequest
*/
func (a *DeviceApiService) AcceptUserCodeRequest(ctx context.Context) ApiAcceptUserCodeRequestRequest {
	return ApiAcceptUserCodeRequestRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return OAuth2RedirectTo
func (a *DeviceApiService) AcceptUserCodeRequestExecute(r ApiAcceptUserCodeRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
	body, err := json.Marshal(r.acceptDeviceUserCodeRequest)
	if err != nil {
		return nil, nil, err
	}

	localBasePath, err := a.client.GetConfig().ServerURLWithContext(r.ctx, "OAuth2APIService.AcceptUserCodeRequest")
	if err != nil {
		return nil, nil, err
	}
	url := localBasePath + "/admin/oauth2/auth/requests/device/accept"

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, reportError("error when constructing request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	query := req.URL.Query()
	query.Add("challenge", *r.deviceChallenge)
	req.URL.RawQuery = query.Encode()

	client := a.client.GetConfig().HTTPClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, reportError("failed to verify device code: %s", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	if resp.StatusCode >= 300 {
		newErr := &APIError{
			body:  respBody,
			error: resp.Status,
		}
		return nil, resp, newErr
	}

	acceptDeviceResponse := new(hClient.OAuth2RedirectTo)
	err = json.Unmarshal(respBody, acceptDeviceResponse)
	if err != nil {
		return nil, resp, reportError("error when parsing request body: %s", err)
	}

	return acceptDeviceResponse, resp, nil
}

func newDeviceApiService(api *hClient.APIClient) *DeviceApiService {
	a := new(DeviceApiService)
	a.client = api
	a.OAuth2API = api.OAuth2API
	return a
}
