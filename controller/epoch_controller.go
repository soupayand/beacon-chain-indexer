package controller

import (
	"context"
	"encoding/json"
	db "go-beacon-chain-indexer/db"
	"go-beacon-chain-indexer/logger"
	"go-beacon-chain-indexer/model"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v4/pgxpool"
)

type EpochController struct {
	db *db.Database
}

func NewEpochController(Pool *pgxpool.Pool) *EpochController {
	return &EpochController{
		db: db.NewDatabase(Pool),
	}
}

/*
This handler is responsible for fetching the indexed epoch data from the databse based as a json output
*/
func (c *EpochController) GetData(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	if len(queryParams) > 1 {
		http.Error(w, "Pass only one attribute for filtering", http.StatusBadRequest)
		return
	}
	var paramName string
	var paramValue string
	for name, values := range queryParams {
		// Ignore parameters without values
		if len(values) == 0 {
			break
		}

		paramValue = values[0]
		paramName = name
	}

	sqlQuery := "SELECT * FROM beacon_chain_data"
	if paramValue != "" {
		sqlQuery += " WHERE " + paramName + " = " + paramValue
		sqlQuery += " ORDER BY " + paramName + " DESC"
	} else {
		sqlQuery += " ORDER BY slot DESC"
	}

	rows, err := c.db.Pool.Query(context.Background(), sqlQuery)
	if err != nil {
		logger.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var data []model.BeaconChainData
	var slot int64
	for rows.Next() {
		var beaconData model.BeaconChainData
		err = rows.Scan(
			&slot,
			&beaconData.Data.Root,
			&beaconData.Data.Canonical,
			&beaconData.Data.Header.Message.ProposerIndex,
			&beaconData.Data.Header.Message.ParentRoot,
			&beaconData.Data.Header.Message.StateRoot,
			&beaconData.Data.Header.Message.BodyRoot,
			&beaconData.Data.Header.Signature,
			&beaconData.Data.UnixTime,
			&beaconData.Epoch,
		)
		beaconData.Data.Header.Message.Slot = strconv.FormatInt(slot, 10)
		if err != nil {
			logger.LogError(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data = append(data, beaconData)
	}
	w.Header().Set("Content-Type", "application/json")

	//Add("Content-Type" : "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		logger.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
