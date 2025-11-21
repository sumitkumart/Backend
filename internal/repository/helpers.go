package repository

import (
	"github.com/shopspring/decimal"
)

// decimalToString converts decimal to string for MongoDB storage
func decimalToString(d decimal.Decimal) string {
	return d.String()
}

// stringToDecimal converts string to decimal from MongoDB retrieval
func stringToDecimal(s string) decimal.Decimal {
	dec, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return dec
}

