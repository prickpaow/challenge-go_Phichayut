package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"go-tamboon/api"
	"go-tamboon/cipher"
	"go-tamboon/models"
)

const (
	maxWorkers           = 5
	delayBetweenRequests = 1 * time.Second
	maxRetries           = 5
	decryptedFile        = "decrypted_results.txt"
	processedFile        = "processed_results.txt"
	summaryFile          = "summary.txt"
)

type apiResult struct {
	Success   bool
	Amount    int64
	DonorName string
	Err       error
}

func worker(donationChan <-chan models.Donation, resultsChan chan<- *apiResult, wg *sync.WaitGroup, stopChan chan struct{}) {
	defer wg.Done()

	for donation := range donationChan {
		select {
		case <-stopChan:
			fmt.Println("Stopping worker due to rate limit.")
			return
		default:
			var success bool
			var err error
			var retries int
			backoff := delayBetweenRequests

			for retries < maxRetries {
				success, err = api.CreateCharge(donation)
				if err == nil {
					break
				}

				if err.Error() == "API rate limit has been exceeded" {
					close(stopChan) // Stop further processing if rate limit is hit
					fmt.Println("API rate limit exceeded. Stopping further API calls.")
					resultsChan <- &apiResult{false, int64(donation.Amount), donation.Name, err}
					return
				}

				retries++
				fmt.Printf("Retry %d for donation %s due to error: %v\n", retries, donation.Name, err)
				time.Sleep(backoff)
				backoff *= 2
			}

			if err != nil {
				resultsChan <- &apiResult{false, int64(donation.Amount), donation.Name, err}
			} else {
				resultsChan <- &apiResult{success, int64(donation.Amount), donation.Name, nil}
				fmt.Printf("Successful donation for %s\n", donation.Name)
			}

			time.Sleep(delayBetweenRequests)
		}
	}
}

func writeResultsToFile(fileName string, data string) {
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("Failed to create %s: %v\n", fileName, err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		fmt.Printf("Failed to write to %s: %v\n", fileName, err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a CSV file as an argument")
		return
	}

	dataFile := os.Args[1]

	file, err := os.Open(dataFile)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	rotReader, err := cipher.NewRot128Reader(file)
	if err != nil {
		fmt.Println("Failed to create ROT-128 reader:", err)
		return
	}

	reader := csv.NewReader(rotReader)
	rows, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Failed to read CSV file:", err)
		return
	}

	if len(rows) <= 1 {
		fmt.Println("No data found in the file")
		return
	}

	var donations []models.Donation
	var decryptedData strings.Builder

	for _, row := range rows[1:] {
		donation, err := models.NewDonation(row)
		if err != nil {
			fmt.Printf("Invalid donation data: %v\n", err)
			continue
		}
		donations = append(donations, donation)
		decryptedData.WriteString(fmt.Sprintf("%v\n", row)) // Write decrypted data
	}

	writeResultsToFile(decryptedFile, decryptedData.String()) // Save decrypted data to file

	var totalReceived, totalSuccessful, totalFaulty int64
	var topDonors []models.Donation
	var wg sync.WaitGroup
	var mu sync.Mutex
	var stopChan = make(chan struct{})

	donationChan := make(chan models.Donation, len(donations))
	resultsChan := make(chan *apiResult, len(donations))

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(donationChan, resultsChan, &wg, stopChan)
	}

	for _, donation := range donations {
		donationChan <- donation
	}
	close(donationChan)

	wg.Wait()
	close(resultsChan)

	var processedData strings.Builder

	for result := range resultsChan {
		mu.Lock()
		totalReceived += result.Amount
		if result.Success {
			totalSuccessful += result.Amount
			topDonors = append(topDonors, models.Donation{Name: result.DonorName, Amount: int(result.Amount)})
			processedData.WriteString(fmt.Sprintf("Success for %s: THB %.2f\n", result.DonorName, float64(result.Amount)/100))
		} else {
			totalFaulty += result.Amount
			processedData.WriteString(fmt.Sprintf("Failed for %s: %v\n", result.DonorName, result.Err))
		}
		mu.Unlock()
	}

	writeResultsToFile(processedFile, processedData.String()) // Save processed data to file

	sort.Slice(topDonors, func(i, j int) bool {
		return topDonors[i].Amount > topDonors[j].Amount
	})

	if len(topDonors) > 3 {
		topDonors = topDonors[:3]
	}

	averageDonation := float64(totalReceived) / float64(len(donations))
	if len(donations) == 0 {
		averageDonation = 0
	}

	var summaryData strings.Builder
	summaryData.WriteString(fmt.Sprintf("Total received: THB %.2f\n", float64(totalReceived)/100))
	summaryData.WriteString(fmt.Sprintf("Successfully donated: THB %.2f\n", float64(totalSuccessful)/100))
	summaryData.WriteString(fmt.Sprintf("Faulty donations: THB %.2f\n", float64(totalFaulty)/100))
	summaryData.WriteString(fmt.Sprintf("Average donation per person: THB %.2f\n", averageDonation/100))
	summaryData.WriteString("Top donors:")
	if len(topDonors) > 0 {
		for _, donor := range topDonors {
			summaryData.WriteString(fmt.Sprintf(" %s (THB %.2f)", donor.Name, float64(donor.Amount)/100))
		}
	} else {
		summaryData.WriteString(" None")
	}
	summaryData.WriteString("\n")

	writeResultsToFile(summaryFile, summaryData.String()) // Save summary to file
}
