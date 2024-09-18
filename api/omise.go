package api

import (
	"fmt"
	"go-tamboon/models"
	"log"
	"time"

	"github.com/omise/omise-go"
	"github.com/omise/omise-go/operations"
)

// CreateToken generates a token from card details
func CreateToken(donation models.Donation) (string, error) {
	client, _ := omise.NewClient("pkey_test_5t18i53q5ceinqzqsff", "skey_test_5t18i566k19xcisl26n")

	result := &omise.Token{}

	err := client.Do(result, &operations.CreateToken{
		Name:            donation.Name,
		Number:          donation.CCNumber,
		ExpirationMonth: time.Month(donation.ExpMonth),
		ExpirationYear:  donation.ExpYear,
		SecurityCode:    donation.CVV,
	})
	if err != nil {
		// Handle different error messages
		if omiseErr, ok := err.(*omise.Error); ok {
			switch {
			case omiseErr.Message == "invalid_card":
				log.Printf("Invalid card error: %s\n", omiseErr.Message)
			case omiseErr.Message == "too_many_requests":
				log.Printf("Rate limit exceeded: %s\n", omiseErr.Message)
			default:
				log.Printf("Unhandled error: %s\n", omiseErr.Message)
			}
		} else {
			log.Printf("Error creating token: %v\n", err)
		}
		return "", fmt.Errorf("failed to create token for %s: %v", donation.Name, err)
	}

	log.Printf("Token created successfully for %s: %s\n", donation.Name, result.ID)
	return result.ID, nil
}

// CreateCharge creates a charge using the donation details
func CreateCharge(donation models.Donation) (bool, error) {
	// Step 1: Generate a token for the card
	token, err := CreateToken(donation)
	if err != nil {
		return false, fmt.Errorf("failed to create token: %v", err)
	}

	client, _ := omise.NewClient("pkey_test_5t18i53q5ceinqzqsff", "skey_test_5t18i566k19xcisl26n")

	result := &omise.Charge{}
	err = client.Do(result, &operations.CreateCharge{
		Amount:   int64(donation.Amount),
		Currency: "thb",
		Card:     token, // Use the token created in Step 1
	})
	if err != nil {
		// Handle different error messages
		if omiseErr, ok := err.(*omise.Error); ok {
			switch {
			case omiseErr.Message == "invalid_card":
				log.Printf("Invalid card error: %s\n", omiseErr.Message)
			case omiseErr.Message == "too_many_requests":
				log.Printf("Rate limit exceeded: %s\n", omiseErr.Message)
			default:
				log.Printf("Unhandled error: %s\n", omiseErr.Message)
			}
		} else {
			log.Printf("Error creating charge: %v\n", err)
		}
		return false, fmt.Errorf("failed to create charge for %s: %v", donation.Name, err)
	}

	log.Printf("Charge created successfully for %s: %v\n", donation.Name, result.ID)
	return true, nil
}
