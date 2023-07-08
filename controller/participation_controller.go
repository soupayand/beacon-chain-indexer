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
	latestEpochNumber, err := p.s.FetchLatestEpochNumber()
	handleInternalServerError(err, w)
	noOfEpochs, _ := strconv.ParseInt(params["epoch"], 10, 64)
	if noOfEpochs == 0 {
		noOfEpochs = 1
	}
	validatorIndex := params["validatorIndex"]
	startingEpochNumber := latestEpochNumber - noOfEpochs + 1
	slotsPerEpoch, _ := strconv.ParseInt(os.Getenv("SLOTS_PER_EPOCH"), 10, 64)
	missed, participated, totalValidators := 0, 0, 0
	for epoch := startingEpochNumber; epoch <= latestEpochNumber; epoch++ {
		if validatorIndex != "" {
			m, p := calculateValidatorParticipationRate(p.s, epoch, validatorIndex)
			missed += m
			participated += p
		} else {
			m, p, t := calculateTotalParticipationRate(p.s, epoch)
			missed += m
			participated += p
			totalValidators += t

		}
	}
	var participation model.Participation
	if validatorIndex != "" {
		participation.ParticipationFactor = float64(missed) / (float64(noOfEpochs) * float64(slotsPerEpoch))
	} else {
		participation.ParticipationFactor = float64(1) - float64(missed)/(float64(noOfEpochs)*float64(slotsPerEpoch)*float64(totalValidators))
	}
	participation.MissedAttestations = missed
	participation.ActualAttestations = participated
	err = json.NewEncoder(w).Encode(participation)
	handleInternalServerError(err, w)
}

func calculateTotalParticipationRate(s *service.Service, epoch int64) (int, int, int) {
	totalValidators := 0
	participated := 0
	missed := 0
	validators := s.FetchTotalNumberOfValidators(epoch)
	startingSlot, _ := s.GetSlotRange(epoch)
	startingSlot = startingSlot + 1
	aggregationBits := s.FetchAggregationBits(startingSlot)
	pr, m := 0, 0
	for index, _ := range aggregationBits {
		validatorsInCommittee := validators[index]
		totalValidators += validatorsInCommittee
		pr, m, _ = missedAttestations(aggregationBits[index], validatorsInCommittee)
		participated += pr
		missed += m
	}

	return missed, participated, totalValidators
}

func calculateValidatorParticipationRate(s *service.Service, epoch int64, validatorIndex string) (int, int) {
	participated := 0
	missed := 0
	committeeIndex, positionInIndex := s.FetchValidatorInfo(epoch, validatorIndex)
	startingSlot, _ := s.GetSlotRange(epoch)
	startingSlot = startingSlot + 1
	aggregationBits := s.FetchAggregationBits(startingSlot)
	for index, _ := range aggregationBits {
		if index == committeeIndex {
			bitValue := getValidatorAttestationBit(aggregationBits[index], positionInIndex)
			if bitValue == "" {
				logger.LogError(errors.New(fmt.Sprintf("Error calculating the validator attestation bit in commiittee", committeeIndex)))
				return 0, 0
			}
			if bitValue == "0" {
				missed += missed
			} else if bitValue == "1" {
				participated += participated
			}
		}
	}
	return missed, participated
}

/*
This function fetches the no of 0s and 1s in aggregation_bits string in a specific slot.
These no of occurrences determine the no of participated and missed attestations
*/
func missedAttestations(aggBitsHex string, validatorSetSize int) (int, int, int) {
	decoded, err := hex.DecodeString(aggBitsHex[2:])
	if err != nil {
		logger.LogError(err)
		return 0, 0, 0

	}
	binary := ""
	for _, b := range decoded {
		binary += fmt.Sprintf("%08b", b)
	}
	binary = binary[:validatorSetSize]
	countZeros := 0
	countOnes := 0
	i := 0
	for _, bit := range binary {
		if bit == '0' {
			countZeros++
		} else if bit == '1' {
			countOnes++
		}
		i++
	}
	return countOnes, countZeros, i
}

func getValidatorAttestationBit(aggBitsHex string, committeePos int) string {
	decoded, err := hex.DecodeString(aggBitsHex[2:])
	if err != nil {
		logger.LogError(err)
		return ""

	}
	binary := ""
	for _, b := range decoded {
		binary += fmt.Sprintf("%08b", b)
	}
	return string(binary[committeePos])
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
