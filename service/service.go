package service

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"go-beacon-chain-indexer/db"
	"go-beacon-chain-indexer/logger"
	"go-beacon-chain-indexer/model"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	GenesisUnixTime = 1606804223
)

type Service struct {
	db *db.Database
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		db: db.NewDatabase(pool),
	}
}

/*
This method loads the epoch indexed data by fetching them from external API and loading them to the database
*/
func (s *Service) Run() error {
	err := s.db.DeleteData()
	if err != nil {
		logger.LogError(err)
		return err
	}
	latestSlot, err := s.fetchLatestSlot()
	if err != nil {
		logger.LogError(err)
		return fmt.Errorf("failed to fetch latest slot: %v", err)
	}

	genesisSlotUnixTimestamp := int64(GenesisUnixTime) // Replace with the actual genesis slot Unix timestamp
	//latestEpoch := getEpochNumber(latestSlot)
	startingSlot := latestSlot
	endSlot := getStartingSlotNumber(latestSlot)
	var wg sync.WaitGroup
	rateLimiter := time.Tick(time.Second / 24)
	for slot := startingSlot; slot >= endSlot; slot-- {
		wg.Add(1)
		go func(slot int64) {
			defer wg.Done()
			timePerSlot, _ := strconv.ParseInt(os.Getenv("SECONDS_PER_SLOT"), 10, 64)
			slotTime := genesisSlotUnixTimestamp + slot*timePerSlot // Assuming slot duration is 384 seconds
			<-rateLimiter
			beaconData, err := s.fetchBeaconData(slot)
			if err != nil || beaconData == nil {
				return
			}
			epoch := getEpochNumber(slot)

			err = s.db.InsertData(epoch, slot, slotTime, beaconData)
			if err != nil {
				logger.LogError(err)
				return
			}
		}(slot)
	}
	wg.Wait()
	log.Println("Data insertion completed!")
	return nil
}

/*
This method fetches the latest finalized slot number
*/
func (s *Service) fetchLatestSlot() (int64, error) {
	url := "https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/headers/finalized" // Replace with the actual endpoint

	response, err := http.Get(url)
	if err != nil {
		logger.LogError(err)
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.LogError(err)
		}
	}(response.Body)

	var beaconData model.BeaconChainData
	err = json.NewDecoder(response.Body).Decode(&beaconData)
	if err != nil {
		logger.LogError(err)
		return 0, err
	}
	slotNumber, _ := strconv.ParseInt(beaconData.Data.Header.Message.Slot, 10, 64)
	return slotNumber, nil
}

/*
This method fetches the header data for a specific slot
*/
func (s *Service) fetchBeaconData(slotNumber int64) (*model.BeaconChainData, error) {
	url := fmt.Sprintf("https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/headers/%v", slotNumber)

	response, err := http.Get(url)
	if err != nil {
		logger.LogError(err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.LogError(err)
		}
	}(response.Body)

	if response.StatusCode == 404 {
		logger.LogInfo("Slot number ", slotNumber, " was missed")
		return nil, nil
	}

	var beaconData model.BeaconChainData
	err = json.NewDecoder(response.Body).Decode(&beaconData)

	return &beaconData, nil
}

/*
This method calculates the epoch number from the slot number
*/
func getEpochNumber(slotNumber int64) int64 {
	var slotPerEpoch, _ = strconv.ParseInt(os.Getenv("SLOTS_PER_EPOCH"), 10, 32)
	epochNumber := slotNumber / slotPerEpoch
	return epochNumber
}

/*
This method determines the starting slot number in in that epoch from the supplied slot number
*/
func getStartingSlotNumber(currentSlotNumber int64) int64 {
	var epochCount, _ = strconv.ParseInt(os.Getenv("EPOCH_COUNT"), 10, 32)
	var slotPerEpoch, _ = strconv.ParseInt(os.Getenv("SLOTS_PER_EPOCH"), 10, 32)
	currentEpoch := currentSlotNumber / slotPerEpoch
	startingEpoch := currentEpoch - epochCount + 1
	startingSlotNumber := startingEpoch * slotPerEpoch
	return startingSlotNumber
}
func (s *Service) GetSlotRange(epoch int64) (int64, int64) {
	startSlot := epoch * 32
	endSlot := startSlot + 32 - 1
	return startSlot, endSlot
}

/*
This method fetches the latest finalized epoch number from the latest finalized slot number
*/
func (s *Service) FetchLatestEpochNumber() (int64, error) {
	slotNumber, err := s.fetchLatestSlot()
	if err != nil {
		return 0, err
	}
	epochNumber := getEpochNumber(slotNumber)
	return epochNumber, nil
}

/*
This method fetches the no of validators in a specific validator set
*/
func (s *Service) FetchTotalNumberOfValidators(epoch int64) map[string]int {
	logger.LogInfo("Fetching committees for epoch %v", epoch)
	indexToValidators := make(map[string]int)
	response, err := http.Get(fmt.Sprintf("https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/states/finalized/committees?epoch=%v", epoch)) // Replace with the actual API endpoint
	if err != nil {
		logger.LogError(err)

	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.LogError(err)
		}
	}(response.Body)

	dec := json.NewDecoder(response.Body)
	var committeeData struct {
		Data []model.Committee `json:"data"`
	}

	err = dec.Decode(&committeeData)
	if err != nil {
		logger.LogError(err)

	}
	for _, committee := range committeeData.Data {
		indexToValidators[committee.Index] = len(committee.Validators)
	}
	return indexToValidators
}

/*
This method fetches the committee and the position of the validator in a specific epoch
*/
func (s *Service) FetchValidatorInfo(epoch int64, validatorIndex string) (string, int) {
	logger.LogInfo("Fetching committee info for validatorIndex %v in epoch %v", validatorIndex, epoch)
	response, err := http.Get(fmt.Sprintf("https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/states/finalized/committees?epoch=%v", epoch)) // Replace with the actual API endpoint
	if err != nil {
		logger.LogError(err)

	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.LogError(err)
		}
	}(response.Body)

	dec := json.NewDecoder(response.Body)
	var committeeData struct {
		Data []model.Committee `json:"data"`
	}

	err = dec.Decode(&committeeData)
	if err != nil {
		logger.LogError(err)

	}
	committeeIndex := ""
	committeePosition := 0
	for _, committee := range committeeData.Data {
		for i, validator := range committee.Validators {
			if validator == validatorIndex {
				committeeIndex = committee.Index
				committeePosition = i
			}
		}
	}
	return committeeIndex, committeePosition
}

/*
This method fetches the aggregation_bits string based on each index in a slot
*/
func (s *Service) FetchAggregationBits(startingSlot int64) map[string]string {
	logger.LogInfo("Fetching attestation bits for %v", startingSlot)
	indexToAggregationBits := make(map[string]string)
	slotPerEpoch, _ := strconv.ParseInt(os.Getenv("SLOTS_PER_EPOCH"), 10, 64)
	endSlot := startingSlot + slotPerEpoch
	for slot := startingSlot; slot < endSlot; slot++ {
		response, err := http.Get(fmt.Sprintf("https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/blocks/%v/attestations", slot)) // Replace with the actual API endpoint
		if err != nil {
			logger.LogError(err)
			continue
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.LogError(err)
			}
		}(response.Body)

		if response.StatusCode == 404 {
			logger.LogInfo("Slot number ", slot, " was missed")
			continue

		}

		dec := json.NewDecoder(response.Body)
		dec.UseNumber()

		var attestationData struct {
			Data []model.Attestation `json:"data"`
		}

		err = dec.Decode(&attestationData)
		if err != nil {
			fmt.Printf("Failed to decode JSON response: %v\n", err)

		}
		for _, attestation := range attestationData.Data {
			indexToAggregationBits[attestation.Details.Index] = attestation.AggregationBits
		}
	}
	return indexToAggregationBits
}
