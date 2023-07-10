package controller

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"go-beacon-chain-indexer/db"
	"go-beacon-chain-indexer/logger"
	"go-beacon-chain-indexer/model"
	"go-beacon-chain-indexer/service"
	"net/http"
	"os"
	"strconv"
)

type ParticipationController struct {
	db *db.Database
	s  *service.Service
}

func NewParticipationController(pool *pgxpool.Pool, service *service.Service) *ParticipationController {
	return &ParticipationController{
		db: db.NewDatabase(pool),
		s:  service,
	}
}

/*
This handler is responsible for fetching the total participation rate of the recent finalized epoch
and sending back response in json
*/
func (p *ParticipationController) GetParticipationRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := parseQueryParameters(w, r)
	if params == nil {
		return
	}
	latestEpochNumberCh := make(chan int64)
	validatorSetSizeCh := make(chan int)
	go func() {
		latestEpochNumber, err := p.s.FetchLatestEpochNumber()
		if err != nil {
			logger.LogError(err)
		}
		latestEpochNumberCh <- latestEpochNumber
	}()
	noOfEpochs, _ := strconv.ParseInt(params["epoch"], 10, 64)
	if noOfEpochs == 0 {
		noOfEpochs = 1
	}
	validatorIndex := params["validatorIndex"]
	latestEpochNumber := <-latestEpochNumberCh
	startingEpochNumber := latestEpochNumber - noOfEpochs + 1
	slotsPerEpoch, _ := strconv.ParseInt(os.Getenv("SLOTS_PER_EPOCH"), 10, 64)
	votingValidators := 0
	missed := 0
	participated := 0
	for epoch := startingEpochNumber; epoch <= latestEpochNumber; epoch++ {
		if validatorIndex != "" {
			m, pr := calculateValidatorParticipationRate(p.s, epoch, validatorIndex)
			missed += m
			participated += pr
		} else {
			m, t := calculateParticipationInEpoch(p.s, epoch)
			missed += m
			votingValidators += t
		}
	}

	participationFactor := float64(1)
	go func() {
		validatorSetSize, err := p.s.FetchValidatorSetSize()
		if err != nil {
			logger.LogError(errors.New("Error fetching finalized validator set size"))
			return
		}
		validatorSetSizeCh <- validatorSetSize
	}()
	validatorSetSize := <-validatorSetSizeCh
	if validatorIndex != "" {
		participationFactor = float64(missed) / (float64(noOfEpochs) * float64(slotsPerEpoch))
	} else if votingValidators > 0 {
		participationFactor = 1 - (float64(missed) / (float64(noOfEpochs) * float64(slotsPerEpoch) * float64(validatorSetSize)))
	}

	participation := model.Participation{
		MissedAttestations:  missed,
		ParticipationFactor: participationFactor,
		ValidatorSetSize:    validatorSetSize,
	}
	err := json.NewEncoder(w).Encode(participation)
	if err != nil {
		handleInternalServerError(err, w)
	}
}

func calculateParticipationInEpoch(s *service.Service, epoch int64) (int, int) {
	validators := s.FetchTotalNumberOfValidators(epoch)
	startingSlot, _ := s.GetSlotRange(epoch)
	startingSlot++
	aggregationBits := s.FetchAggregationBits(startingSlot)
	missedAttestations := 0
	totalVotingValidators := 0
	for index, bits := range aggregationBits {
		validatorSetSize := validators[index]
		totalVotingValidators += validatorSetSize
		m := calculateMissedAttestationCount(bits, validatorSetSize)
		missedAttestations += m
	}
	return missedAttestations, totalVotingValidators
}

func calculateValidatorParticipationRate(s *service.Service, epoch int64, validatorIndex string) (int, int) {
	committeeIndex, positionInIndex := s.FetchValidatorInfo(epoch, validatorIndex)
	startingSlot, _ := s.GetSlotRange(epoch)
	startingSlot++
	aggregationBits := s.FetchAggregationBits(startingSlot)
	missed := 0
	participated := 0
	for index, bits := range aggregationBits {
		if index == committeeIndex {
			bitValue := getValidatorAttestationBit(bits, positionInIndex)
			if bitValue == "" {
				logger.LogError(errors.New(fmt.Sprintf("Error calculating the validator attestation bit in committee %d", committeeIndex)))
				return 0, 0
			}
			if bitValue == "0" {
				missed++
			} else if bitValue == "1" {
				participated++
			}
		}
	}

	return missed, participated
}

/*
This function fetches the no of 0s and 1s in aggregation_bits string in a specific slot.
These no of occurrences determine the no of participated and missed attestations
*/
func calculateMissedAttestationCount(aggBitsHex string, validatorSetSize int) int {
	decoded, err := hex.DecodeString(aggBitsHex[2:])
	if err != nil {
		logger.LogError(err)
		return 0
	}
	countZeros := 0
	for i := 0; i < validatorSetSize; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bit := (decoded[byteIndex] >> (7 - bitIndex)) & 0x01
		if bit == 0 {
			countZeros++
		}
	}

	return countZeros
}

func getValidatorAttestationBit(aggBitsHex string, committeePos int) string {
	decoded, err := hex.DecodeString(aggBitsHex[2:])
	if err != nil {
		logger.LogError(err)
		return ""
	}
	byteIndex := committeePos / 8
	bitIndex := committeePos % 8
	if byteIndex >= len(decoded) {
		return ""
	}
	bitValue := (decoded[byteIndex] >> (7 - bitIndex)) & 0x01
	return strconv.Itoa(int(bitValue))
}

func parseQueryParameters(w http.ResponseWriter, r *http.Request) map[string]string {
	queryParams := r.URL.Query()
	if len(queryParams) > 2 {
		http.Error(w, "Only two parameters are allowed : epoch and validator index", http.StatusBadRequest)
		return nil
	} else if len(queryParams) < 1 {
		http.Error(w, "Epoch query parameter is a must", http.StatusBadRequest)
		return nil
	}
	params := make(map[string]string)
	for name, values := range queryParams {
		if len(values) == 0 {
			break
		}
		params[name] = values[0]
	}
	if params["epoch"] == "" {
		http.Error(w, "Epoch query parameter is a must and it must be in this format: epoch=$val", http.StatusBadRequest)
		return nil
	}
	return params
}

func handleInternalServerError(err error, w http.ResponseWriter) {
	if err != nil {
		logger.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
