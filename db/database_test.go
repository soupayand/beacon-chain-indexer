package db

/*
import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"os"
	"testing"
)

func TestInsertData(t *testing.T) {
	err := godotenv.Load()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock db: %v", err)
	}
	defer db.Close()

	// Create a test db connection pool using the mock DB
	pool, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL")) // Use any connection string
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	// Set the mock expectations
	mock.ExpectExec("INSERT INTO test_beacon_chain_data").WithArgs(
		6814921, // slot
		"0x09933d7c620afa41a1fc6c3026bf2ba025bc3e2d38315d55cbe9a84000174198", // root
		true,     // canonical
		"189931", // proposer_index
		"0x09933d7c620afa41a1fc6c3026bf2ba025bc3e2d38315d55cbe9a84000174198", // parent_root
		"0xf1acf0533e9019689b4f03188666e60c2db61865599713a6ba40e8913ba95206", // state_root
		"0x142dc7d16fa72a10d4d292c8b3d485242a8b2b8b506d6a270d69327b3da21500", // body_root
		"0xb65365ca999e5e39b373111e771be843da4dbba492c15890c09a71d4dc1dbd19febd5f9000279ce60dcc8ed11b5965550241c2a2f1a8733c336336898ec0def5555c80b689f785a1619bfae46f648bbbac87c4a746a41ea2642c92083775bde1", // signature
		1688583275,
		212966,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	// Create a new db instance with the mock pool
	db := NewDatabase(pool)

	// Insert test data
	err = db.InsertData(212966, 6814920, 1688583263, &BeaconChainData{})
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	// Check the mock expectations
	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func TestDeleteData(t *testing.T) {
	// Create a new mock db connection and mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock db: %v", err)
	}
	defer db.Close()

	// Create a test db connection pool using the mock DB
	pool, err := pgxpool.Connect(context.Background(), "mockdb") // Use any connection string
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	// Set the mock expectations
	mock.ExpectExec("DELETE FROM beacon_chain_data").WillReturnResult(sqlmock.NewResult(1, 1))

	// Create a new db instance with the mock pool
	db := NewDatabase(pool)

	// Delete the data
	err = db.DeleteData()
	if err != nil {
		t.Errorf("Failed to delete data: %v", err)
	}

	// Check the mock expectations
	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}
*/
