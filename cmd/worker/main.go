package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/francescomari/mongo-worker/internal/task"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGO_URI not found")
	}

	mongoDatabase := os.Getenv("MONGO_DATABASE")
	if mongoDatabase == "" {
		return fmt.Errorf("MONGO_DATABASE not found")
	}

	database := task.Database{
		URI:      mongoURI,
		Database: mongoDatabase,
	}

	if err := mongoConnect(&database); err != nil {
		return err
	}

	log.Printf("connected to MongoDB")

	defer func() {
		if err := mongoDisconnect(&database); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-signalCh:
			return nil
		case <-time.After(1 * time.Second):
			// Process next task.
		}

		if err := processTask(&database); err != nil {
			log.Printf("process task: %v", err)
		}
	}
}

func mongoConnect(db *task.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return db.Connect(ctx)
}

func mongoDisconnect(db *task.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return db.Disconnect(ctx)
}

func processTask(db *task.Database) error {
	startCtx, startCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer startCancel()

	task, err := db.StartTask(startCtx)
	if err != nil {
		return fmt.Errorf("start task: %v", err)
	}
	if task == nil {
		return nil
	}

	data, err := json.Marshal(task.Data)
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}

	log.Printf("found task %v created at %v with data %v", task.ID, task.CreatedAt, string(data))

	time.Sleep(10 * time.Second)

	finishCtx, finishCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer finishCancel()

	if err := db.FinishTask(finishCtx, task.ID); err != nil {
		return fmt.Errorf("complete task: %v", err)
	}

	return nil
}
