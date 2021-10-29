package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// These constants provide a set of private enum values representing web service status
const (
	idleStatus = iota
	busyStatus
	errorStatus
	notRunningStatus
)

// HealthCheck is a struct for representing the health status of the web service.
type HealthCheck struct {
	Status int `json:"status"`
}

// HealthCheckHandler provides an http response representing the health status of the web service.
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resultBytes, err := json.Marshal(HealthCheck{Status: idleStatus})
		if err != nil {
			fmt.Println(fmt.Errorf("could not convert object to JSON: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resultJSON := string(resultBytes)
		fmt.Println(resultJSON)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, resultJSON)
	default:
		fmt.Println(fmt.Errorf("HealthCheckHandler only supports GET requests"))
	}
}

// main is the primary executor for this executable package.
// It will set up the routes and associate them with their respective handler functions.
// It also starts the http listener and logs an error if it terminates.
func main() {
	http.HandleFunc("/health-check", HealthCheckHandler)
	fmt.Println(fmt.Errorf("web server error: %v", http.ListenAndServe(":8080", nil)))
}
