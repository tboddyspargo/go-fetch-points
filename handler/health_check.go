package handler

import (
	"net/http"

	"github.com/tboddyspargo/fetch/log"
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
		respondWithJSON(w, http.StatusOK, HealthCheck{Status: idleStatus})
	default:
		log.Error("HealthCheckHandler only supports GET requests")
	}
}
