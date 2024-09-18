package models

import (
	"fmt"
	"strconv"
)

// Donation represents a donation record
type Donation struct {
	Name      string
	CCNumber  string
	CVV       string
	ExpMonth  int
	ExpYear   int
	Amount    int    // Amount in subunits
	CardToken string // Token generated from the card details
}

func NewDonation(row []string) (Donation, error) {
	if len(row) < 6 {
		return Donation{}, fmt.Errorf("invalid row length")
	}

	expMonth, err := strconv.Atoi(row[4])
	if err != nil {
		return Donation{}, fmt.Errorf("invalid expiration month: %v", err)
	}

	expYear, err := strconv.Atoi(row[5])
	if err != nil {
		return Donation{}, fmt.Errorf("invalid expiration year: %v", err)
	}

	amount, err := strconv.Atoi(row[1])
	if err != nil {
		return Donation{}, fmt.Errorf("invalid amount: %v", err)
	}

	donation := Donation{
		Name:     row[0],
		CCNumber: row[2],
		CVV:      row[3],
		ExpMonth: expMonth,
		ExpYear:  expYear,
		Amount:   amount,
	}

	return donation, nil
}
