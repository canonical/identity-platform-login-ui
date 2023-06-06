package health

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const okValue = "ok"
const kratosEnvar = "KRATOS_PUBLIC_URL"
const hydraEnvar = "HYDRA_ADMIN_URL"

var aliveSingleton Status
var readySingleton Status

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func getAlive() Status {
	if aliveSingleton.Status == EmptyStatus().Status {
		aliveSingleton = Status{Status: okValue}
	}
	return aliveSingleton
}

func getReady() Status {
	if readySingleton.Status != okValue {
		isReady, msg := readinessChecker()
		if isReady {
			setReady()
		} else {
			setUnReady(msg)
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

func HandleAlive(w http.ResponseWriter, r *http.Request) {
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

func HandleReady(w http.ResponseWriter, r *http.Request) {
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

func readinessChecker() (bool, string) {
	result := true
	errorMessage := "Error:"
	if kratosURL := os.Getenv(kratosEnvar); kratosURL == "" {
		errorMessage = fmt.Sprintf("%s Kratos endpoint not set.", errorMessage)
		result = false
	}
	if hydraURL := os.Getenv(hydraEnvar); hydraURL == "" {
		errorMessage = fmt.Sprintf("%s Hydra endpoint not set.", errorMessage)
		result = false
	}

	if result {
		return result, ""
	}
	return result, errorMessage
}

func TestSetUnalive(msg string) {
	setUnAlive(msg)
}

func TestResetHealth() {
	aliveSingleton = Status{}
	readySingleton = Status{}
}
