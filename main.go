package main

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"go-beacon-chain-indexer/controller"
	"go-beacon-chain-indexer/logger"
	"go-beacon-chain-indexer/service"
	"net/http"
	"os"
	"strconv"
)

func main() {
	logger.InitLogger()
	logger.LogInfo("Server Starting...........................................................")
	err := godotenv.Load()
	if err != nil {
		logger.LogError(err)
	}

	connConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.LogError(err)
	}
	maxConnections, _ := strconv.ParseInt(os.Getenv("MAX_CONNECTIONS"), 10, 32) // Adjust the maximum number of open connections
	connConfig.MaxConns = int32(maxConnections)
	pool, err := pgxpool.ConnectConfig(context.Background(), connConfig)
	if err != nil {
		logger.LogError(err)
	}
	defer pool.Close()

	var s = service.NewService(pool)
	go func() {
		logger.LogInfo("Starting data load service for fetching last 5 epoch data")
		s.Run()
		logger.LogInfo("Loaded data for last 5 epoch")
	}()

	epochController := controller.NewEpochController(pool)
	participationController := controller.NewParticipationController(pool, s)

	http.HandleFunc("/data", epochController.GetData)
	http.HandleFunc("/participation-rate", participationController.GetParticipationRate)
	logger.LogInfo("Starting server at port %v", os.Getenv("PORT"))
	logger.LogError(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
	logger.LogInfo("Server exited and released port %v", os.Getenv("PORT"))
}
