DROP TABLE IF EXISTS beacon_chain_data;

CREATE TABLE IF NOT EXISTS beacon_chain_data ( slot BIGINT NOT NULL, root TEXT NOT NULL, canonical BOOLEAN NOT NULL, proposer_index TEXT NOT NULL, parent_root TEXT NOT NULL, state_root TEXT NOT NULL, body_root TEXT NOT NULL, signature TEXT NOT NULL, unix_time BIGINT NOT NULL, epoch BIGINT NOT NULL,
PRIMARY KEY (slot, unix_time));

SELECT create_hypertable('beacon_chain_data', 'unix_time', chunk_time_interval => 384, partitioning_column => 'slot', number_partitions => 32);

CREATE TABLE IF NOT EXISTS test_beacon_chain_data ( slot BIGINT NOT NULL, root TEXT NOT NULL, canonical BOOLEAN NOT NULL, proposer_index TEXT NOT NULL, parent_root TEXT NOT NULL, state_root TEXT NOT NULL, body_root TEXT NOT NULL, signature TEXT NOT NULL, unix_time BIGINT NOT NULL, epoch BIGINT NOT NULL,
PRIMARY KEY (slot, unix_time));

SELECT create_hypertable('test_beacon_chain_data', 'unix_time', chunk_time_interval => 384, partitioning_column => 'slot', number_partitions => 32);
