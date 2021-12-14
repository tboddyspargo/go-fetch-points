package points

// PayerTotals is an alias for a map with payer name keys and their respective point totals. It provides more efficient lookup and update speeds
type PayerTotals map[string]int32

// PayerBalance is a struct for storing the number of points associated with a payer. An array of these is the expected return type of several API routes.
type PayerBalance struct {
	Payer  string `json:"payer"`
	Points int32  `json:"points"`
}

// PayerTotalsToPayerBalances converts a PayerTotals map to a slice of PayerBalance objects, which is what the web service is expected to return.
func (pt PayerTotals) ToPayerBalances() []PayerBalance {
	var result = []PayerBalance{}
	for k, v := range pt {
		result = append(result, PayerBalance{Payer: k, Points: v})
	}
	return result
}

// GetPayerTotals returns a PayerTotal object representing the current balance for each payer.
func GetPayerTotals() (PayerTotals, error) {
	return payerTotals, nil
}

// TotalAvailable returns the sum of all points for all payers.
func TotalAvailable() (int32, error) {
	var total int32 = 0
	var pt, err = GetPayerTotals()
	if err != nil {
		return total, err
	}
	for _, v := range pt {
		total += v
	}
	return total, nil
}
