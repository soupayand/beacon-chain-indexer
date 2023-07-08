package db

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"go-beacon-chain-indexer/logger"
	"go-beacon-chain-indexer/model"
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewDatabase(pool *pgxpool.Pool) *Database {
	return &Database{
		Pool: pool,
	}
}

/*
This method stores epoch data in the table beacon_chain_data in the db
*/
func (db *Database) InsertData(epoch int64, slot int64, slotTime int64, beaconData *model.BeaconChainData) error {
	_, err := db.Pool.Exec(context.Background(),
		"INSERT INTO beacon_chain_data (slot, epoch, unix_time, root, canonical, proposer_index, parent_root, state_root, body_root, signature) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		slot,
		epoch,
		slotTime,
		beaconData.Data.Root,
		beaconData.Data.Canonical,
		beaconData.Data.Header.Message.ProposerIndex,
		beaconData.Data.Header.Message.ParentRoot,
		beaconData.Data.Header.Message.StateRoot,
		beaconData.Data.Header.Message.BodyRoot,
		beaconData.Data.Header.Signature,
	)
	if err != nil {
		logger.LogError(err)
		return err
	}
	return nil
}

func (db *Database) DeleteData() error {
	_, err := db.Pool.Exec(context.Background(), "DELETE FROM beacon_chain_data")
	if err != nil {
		logger.LogError(err)
		return err
	}
	return nil
}
