package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"
)

const okValue = "ok"

var aliveSingleton Status
var readySingleton Status
var kratos *kratos_client.APIClient
var hydra *hydra_client.APIClient
var apiClientSet bool = false

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func SetApiClients(k *kratos_client.APIClient, h *hydra_client.APIClient) {
	kratos = k
	hydra = h
	apiClientSet = true
}

func getAlive() Status {
	if aliveSingleton.Status == "" {
		aliveSingleton = Status{Status: okValue}
	}
	return aliveSingleton
}

func getReady() Status {
	if readySingleton.Status != okValue {
		isReady := readinessChecker()
		if isReady {
			setReady()
		} else {
			setUnReady("Ory backend have not been confirmed to be available")
		}
	}
	return readySingleton
}

func setUnAlive(msg string) {
	aliveSingleton = Status{
		Status:  http.StatusText(503),
		Message: msg,
	}
}

func setUnReady(msg string) {
	readySingleton = Status{
		Status:  http.StatusText(503),
		Message: msg,
	}
}

func setReady() {
	readySingleton = Status{Status: okValue}
}

func GetAliveHandler() (func(w http.ResponseWriter, r *http.Request), error) {
	if apiClientSet {
		return handleAlive, nil
	}
	return nil, errors.New("API Clients not set in health")
}

func GetReadyHandler() (func(w http.ResponseWriter, r *http.Request), error) {
	if apiClientSet {
		return handleReady, nil
	}
	return nil, errors.New("API Clients not set health")
}

func handleAlive(w http.ResponseWriter, r *http.Request) {
	status := getAlive()
	w.Header().Set("Content-Type", "application/json")
	if status.Status == okValue {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
	body, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error during Health Check Liveness Check\nerror: %s", err.Error())
	}
	w.Write(body)
	return
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	status := getReady()
	w.Header().Set("Content-Type", "application/json")
	if status.Status == okValue {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
	body, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error during Health Check Readiness Check\nerror: %s", err.Error())
	}
	w.Write(body)
	return
}

func EmptyStatus() *Status {
	return &Status{}
}

func readinessChecker() bool {
	_, r, err := kratos.MetadataApi.IsReady(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling Kratos with `MetadataApi.IsReady``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return false
	} else if r.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Error when calling Kratos with `MetadataApi.IsReady``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return false
	}

	_, r, err = hydra.MetadataApi.IsReady(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling Hydra with `MetadataApi.IsReady``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return false
	} else if r.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Error when calling Hydra with `MetadataApi.IsReady``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return false
	}
	return true
}

func TestSetUnalive(msg string) {
	setUnAlive(msg)
}

func TestHandleAlive(w http.ResponseWriter, r *http.Request) {
	handleAlive(w, r)
}

func TestHandleReady(w http.ResponseWriter, r *http.Request) {
	handleReady(w, r)
}

func TestResetHealth() {
	aliveSingleton = Status{}
	readySingleton = Status{}
	kratos = nil
	hydra = nil
	apiClientSet = false
}
