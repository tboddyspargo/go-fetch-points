package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/tboddyspargo/fetch/log"
	"github.com/tboddyspargo/fetch/points"
)

// respondWithJSON is a convenience function that writes to an http.ResponseWriter with JSON output and sets a given StatusCode in the Header.
func respondWithJSON(w http.ResponseWriter, statusCode int, content interface{}) {
	log.Info(content)
	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response, _ = json.Marshal(map[string]string{"errors": err.Error()})
		log.Errorf("unable to convert content to json: content %v; error %v", content, err)
		w.Write(response)
	}
	w.WriteHeader(statusCode)
	w.Write(response)
}

// AddTransactionHandler provides http action for creating new Transaction records.
// The body of the request is expected to contain the relevant fields for a Transaction object.
// All Transactions created by this route are expected to represent points coming from a payer, not initiated by a user.
func AddTransactionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		defer r.Body.Close()
		var t points.Transaction
		body, _ := io.ReadAll(r.Body)
		log.Infof("AddTransactionHandler(): received request: %v", string(body))

		// Populate the transaction object (t) from the body of the request.
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&t); err != nil {
			log.Error(err)
			respondWithJSON(w, http.StatusBadRequest, map[string]string{"errors": err.Error()})
			return
		}

		if err := t.Save(); err != nil {
			log.Error(err)
			respondWithJSON(w, http.StatusBadRequest, map[string]string{"errors": err.Error()})
			return
		}
		respondWithJSON(w, http.StatusCreated, t)
	default:
		methodErr := "AddTransactionHandler only supports POST requests"
		log.Error(methodErr)
		respondWithJSON(w, http.StatusMethodNotAllowed, struct{}{})
	}
}

// PayerPointsHandler provides an http response in the form of a JSON object representing the total points for every payer.
func PayerPointsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		pt, _ := points.GetPayerTotals()
		respondWithJSON(w, http.StatusOK, pt.ToPayerBalances())
	default:
		methodErr := "PayerPointsHandler only supports GET requests"
		log.Error(methodErr)
		respondWithJSON(w, http.StatusMethodNotAllowed, struct{}{})
	}
}

// SpendPointsHandler provides an http action for spending a specified number of points.
// The body of the request is expected to contain a "points" attribute indicating how many points the user would like to spend.
// Points will be spent in order of oldest to most recent and points will not be spent if doing so would bring the balance associated with a particular payer below zero.
// The response will be in the form of a JSON array containing objects representing how many points were used from each payer to satisfy the request.
func SpendPointsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var desiredSpend points.SpendRequest
		err := json.NewDecoder(r.Body).Decode(&desiredSpend)
		if err != nil {
			log.Errorf("unable to parse json: %v", err)
			respondWithJSON(w, http.StatusBadRequest, map[string]string{"errors": err.Error()})
			return
		}

		totalAvailable, _ := points.TotalAvailable()
		if totalAvailable < desiredSpend.Points {
			spendErr := fmt.Errorf("insufficient points. requested: %v; available: %v", desiredSpend.Points, totalAvailable)
			log.Error(spendErr)
			respondWithJSON(w, http.StatusBadRequest, map[string]string{"errors": spendErr.Error()})
			return
		}

		// Sort the transactions in order of oldest to newest.
		transactions, _ := points.GetTransactions()
		sort.Sort(points.ByTimestamp(transactions))

		// Keep track of how many points are spent from each payer to satisfy this request.
		spentPayerPoints := points.PayerTotals{}

		var remainingToSpend int32 = desiredSpend.Points
		for _, t := range transactions {
			// If all requested points have been spent, we're done
			if remainingToSpend <= 0 {
				break
			}

			// Attempt to spend the points from this transaction.
			currentSpent, spendErr := t.SpendPoints(remainingToSpend)
			if spendErr != nil {
				continue
			}
			spentPayerPoints[t.Payer] -= currentSpent
			remainingToSpend -= currentSpent
		}
		result := spentPayerPoints.ToPayerBalances()
		respondWithJSON(w, http.StatusOK, result)
	default:
		methodErr := "SpendPointsHandler only supports POST requests"
		log.Error(methodErr)
		respondWithJSON(w, http.StatusMethodNotAllowed, struct{}{})
	}
}
