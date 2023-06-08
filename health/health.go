package health

//Notes on Thread safety: Currently we only expose methods that read the state of the singletons therefore thread safety is not an issue.
//In case the scope of use case for Health expands: Implement RWLocks for the singleton variables, or implement using buffered channels to change state

import (
	"encoding/json"
	"log"
	"net/http"
)

const okValue = "ok"

var aliveSingleton Status
var readySingleton Status

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func getAlive() Status {
	if aliveSingleton.Status == EmptyStatus().Status {
		SetAlive()
	}
	return aliveSingleton
}

func getReady() Status {
	if readySingleton.Status == EmptyStatus().Status {
		SetReady()
	}
	return readySingleton
}

func SetUnAlive(msg string) {
	aliveSingleton = Status{
		Status:  http.StatusText(503),
		Message: msg,
	}
}

func SetUnReady(msg string) {
	readySingleton = Status{
		Status:  http.StatusText(503),
		Message: msg,
	}
}

func SetReady() {
	readySingleton = Status{Status: okValue}
}

func SetAlive() {
	aliveSingleton = Status{Status: okValue}
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

func TestResetHealth() {
	aliveSingleton = Status{}
	readySingleton = Status{}
}
