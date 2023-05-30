package health

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
	if aliveSingleton.Status == "" {
		aliveSingleton = Status{Status: okValue}
	}
	return aliveSingleton
}

func getReady() Status {
	if readySingleton.Status == "" {
		readySingleton = Status{Status: okValue}
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
		log.Printf("Error during Health Check Liveness Check\nerror: %s", err.Error())
	}
	w.Write(body)
	return
}

func EmptyStatus() *Status {
	return &Status{}
}
