package ServerError

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type GenericError struct {
	Code    int64  `json:"code"`
	Debug   string `json:"debug"`
	Id      string `json:"id"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Request string `json:"request"`
	Status  string `json:"status"`
}

func GenericErrorConstructor(testname string) GenericError {
	error := GenericError{
		Code:    599,
		Debug:   fmt.Sprintf("Test Debug message from %s handler.", testname),
		Id:      fmt.Sprintf("Test-ID-%s", testname),
		Message: fmt.Sprintf("Test Message from %s handler", testname),
		Reason:  fmt.Sprintf("Test Reason from %s handler", testname),
		Request: fmt.Sprintf("Test Request ID"),
		Status:  http.StatusText(500),
	}
	return error
}

func CreateHandlerWithError(testname string) func(w http.ResponseWriter, r *http.Request) {
	e := GenericErrorConstructor(testname)
	ret := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(e.Code))
		jsonResp, err := json.Marshal(e)
		if err != nil {
			log.Printf("Bug in test: %s\nerror: %s", testname, err.Error())
		}
		w.Write(jsonResp)
		return
	}
	return ret
}

func TimeoutHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusGatewayTimeout)
	return
}
