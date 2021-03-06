package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	_ "github.com/joho/godotenv/autoload"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func scheduleJobs(dynamoSession *dynamodb.DynamoDB, logger *zap.Logger) {
	cronLogger := cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))

	// Start the scheduler to fetch latest Facebook posts
	fbFetchFrequency := GetEnv("FB_FETCH_FREQUENCY", "300")
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger)))
	_, err := c.AddFunc(fmt.Sprintf("@every %ss", fbFetchFrequency), func() {
		fetchLatestError := FetchLatestPosts(dynamoSession, logger)
		if fetchLatestError != nil {
			logger.Error("Fetching latest posts failed", zap.Error(fetchLatestError))
		}
	})
	if err != nil {
		logger.Fatal("Unable to start FetchLatestPosts scheduler", zap.Error(err))
	}

	// Start the scheduler to dispatch posts to C-3PO
	dispatcherFrequency := GetEnv("DISPATCHER_FREQUENCY", "150")
	_, err = c.AddFunc(fmt.Sprintf("@every %ss", dispatcherFrequency), func() {
		dispatchError := DispatchFreshPosts(dynamoSession, logger)
		if dispatchError != nil {
			logger.Error("Dispatching fresh posts failed", zap.Error(dispatchError))
		}
	})
	if err != nil {
		logger.Fatal("Unable to start DispatchFreshPosts scheduler", zap.Error(err))
	}
	c.Start()
}

func main() {
	// Logger setup
	logger := GetLogger()
	defer func() {
		err := logger.Sync()
		if err != nil {
			logger.Warn("Unable to gracefully flush buffered log entries", zap.Error(err))
		}
	}()

	// Create a DynamoDB session
	dynamoSession, err := InitializeDynamoSession(logger)
	if err != nil {
		logger.Fatal("Error initializing Dynamo Session", zap.Error(err))
	}
	logger.Debug("Created dynamoDB session", zap.Any("dynamoSession", dynamoSession))

	// Schedule loggers
	scheduleJobs(dynamoSession, logger)

	// Start API server
	initializeAPIServer(logger)
}
