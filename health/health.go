package health

import (
	"encoding/json"
	"net/http"
)

const okValue = "ok"

type Status struct {
	Status string `json:"status"`
}

func HandleAlive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	body, _ := json.Marshal(Status{Status: okValue})
	w.Write(body)
	return
}
