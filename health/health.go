package health

import (
	"encoding/json"
	"log"
	"net/http"
)

const okValue = "ok"

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func GetNotOKStatus(msg string) Status {
	return Status{
		Status:  http.StatusText(http.StatusServiceUnavailable),
		Message: msg,
	}
}

func GetOKStatus() Status {
	return Status{Status: okValue}
}

func HandleAlive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	body, err := json.Marshal(GetOKStatus())
	if err != nil {
		log.Printf("Error during Health Check Liveness Check\nerror: %s", err.Error())
	}
	w.Write(body)
	return
}

func HandleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	body, err := json.Marshal(GetOKStatus())
	if err != nil {
		log.Printf("Error during Health Check Liveness Check\nerror: %s", err.Error())
	}
	w.Write(body)
	return
}

func EmptyStatus() *Status {
	return &Status{}
}
