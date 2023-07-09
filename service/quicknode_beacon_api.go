package service

import (
	"encoding/json"
	"errors"
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
	db     *db.Database
	client *http.Client
}

type Counter struct {
	value int
	mutex sync.Mutex
}

func NewCounter() *Counter {
	return &Counter{}
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		db: db.NewDatabase(pool),
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,               // Set the maximum number of idle connections in the pool
				IdleConnTimeout:     30 * time.Second, // Set the maximum idle connection timeout
				MaxIdleConnsPerHost: 10,               // Set the maximum number of idle connections per host
			},
		},
	}
}

/*
This method loads the epoch indexed data by fetching them from external API and loading them to the database
*/
func (s *Service) Run() {
	err := indexEpochData(s)
	if err != nil {
		logger.LogError(errors.New("Error encountered while indexing epoch data from quicnode api"))
	}
}

func indexEpochData(s *Service) error {
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

	response, err := s.client.Get(url)
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
	response, err := s.client.Get(url)
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
	response, err := s.client.Get(fmt.Sprintf("https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/states/finalized/committees?epoch=%v", epoch))
	if err != nil {
		logger.LogError(err)
		return nil
	}
	defer response.Body.Close()
	dec := json.NewDecoder(response.Body)
	dec.UseNumber()
	committeeData := make(map[string]interface{})
	err = dec.Decode(&committeeData)
	if err != nil {
		logger.LogError(err)
		return nil
	}
	data, ok := committeeData["data"].([]interface{})
	if !ok {
		logger.LogError(errors.New("data field is not an array"))
		return nil
	}
	for _, item := range data {
		committee, ok := item.(map[string]interface{})
		if !ok {
			logger.LogError(errors.New("invalid committee object"))
			continue
		}
		index, ok := committee["index"].(string)
		if !ok {
			logger.LogError(errors.New("invalid index value"))
			continue
		}
		validators, ok := committee["validators"].([]interface{})
		if !ok {
			logger.LogError(errors.New("invalid validators value"))
			continue
		}
		indexToValidators[index] = len(validators)
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
	indexToAggregationBits := sync.Map{} // Thread-safe map
	slotPerEpoch, _ := strconv.ParseInt(os.Getenv("SLOTS_PER_EPOCH"), 10, 64)
	endSlot := startingSlot + slotPerEpoch
	rateLimiter := time.Tick(time.Second / 25)
	var wg sync.WaitGroup
	for slot := startingSlot; slot < endSlot; slot++ {
		<-rateLimiter
		wg.Add(1)
		go func(slot int64) {
			defer wg.Done()
			response, err := s.client.Get(fmt.Sprintf("https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/blocks/%v/attestations", slot)) // Replace with the actual API endpoint
			if err != nil {
				logger.LogError(err)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					logger.LogError(err)
				}
			}(response.Body)

			if response.StatusCode == 404 {
				logger.LogInfo("Slot number ", slot, " was missed")
				return
			}

			dec := json.NewDecoder(response.Body)
			dec.UseNumber()

			var attestationData struct {
				Data []model.Attestation `json:"data"`
			}

			err = dec.Decode(&attestationData)
			if err != nil {
				logger.LogError(err)
				return
			}

			for _, attestation := range attestationData.Data {
				indexToAggregationBits.Store(attestation.Details.Index, attestation.AggregationBits)
			}
		}(slot)
	}

	go func() {
		wg.Wait()
	}()

	result := make(map[string]string)
	indexToAggregationBits.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(string)
		return true
	})
	return result
}

func (s *Service) FetchValidatorSetSize() (int, error) {
	url := "https://wiser-side-morning.discover.quiknode.pro/eth/v1/beacon/states/head/validators?status=active_ongoing" // Replace with the actual API endpoint
	response, err := http.Get(url)
	if err != nil {
		logger.LogError(err)
		return 0, err
	}
	defer response.Body.Close()
	var wg sync.WaitGroup
	counter := NewCounter()
	decoder := json.NewDecoder(response.Body)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			logger.LogError(err)
			return 0, err
		}

		if tok == "data" {
			break
		}
	}
	var data []interface{}
	err = decoder.Decode(&data)
	if err != nil {
		logger.LogError(err)
		return 0, err
	}
	for _, obj := range data {
		wg.Add(1)
		go func(obj interface{}) {
			defer wg.Done()

			_, ok := obj.(map[string]interface{})
			if !ok {
				return
			}
			counter.Increment()
		}(obj)
	}
	wg.Wait()
	return counter.Value(), nil
}

// Increment increments the counter by 1
func (c *Counter) Increment() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value++
}

// Value returns the current value of the counter
func (c *Counter) Value() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.value
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
