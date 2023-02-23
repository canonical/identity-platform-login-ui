package Errors

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type TestErrorReport struct {
	Id         string    `json:"id"`
	Error      TestError `json:"error"`
	Created_at string    `json:"created_at"`
	Updated_at string    `json:"updated_at"`
}

type TestError struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func SelfServiceErrorsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	e := TestError{
		Code:    400,
		Status:  "Bad Request",
		Reason:  "This is a test",
		Message: "This is a test",
	}
	et := TestErrorReport{
		Id:         id,
		Error:      e,
		Created_at: time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Updated_at: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	response, err := json.Marshal(et)
	if err != nil {
		log.Printf("Bug in test: SelfServiceErrorsHandler\nerror: %s", err.Error())
	}
	w.Write(response)
	return
}
