package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/francescomari/mongo-worker/internal/task"
	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		return fmt.Errorf("API_URL not found")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
			// Submit a task.
		}

		if err := submitTask(ctx, apiURL); err != nil {
			log.Printf("submit task: %v", err)
		}
	}
}

func submitTask(ctx context.Context, url string) error {
	data := map[string]interface{}{
		"random": uuid.NewString(),
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/tasks", url), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %v", err)
	}

	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("invalid status code: %v", res.StatusCode)
	}

	var task task.Task

	if err := json.NewDecoder(res.Body).Decode(&task); err != nil {
		return fmt.Errorf("decode: %v", err)
	}

	log.Printf("submitted task %s", task.ID)

	if err := waitForTask(ctx, url, task.ID); err != nil {
		return fmt.Errorf("wait for task: %v", err)
	}

	return nil
}

func waitForTask(ctx context.Context, url, id string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Check the status of the task.
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/tasks/%s", url, id), nil)
		if err != nil {
			return fmt.Errorf("create request: %v", err)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("perform request: %v", err)
		}

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("invalid status code: %v", res.StatusCode)
		}

		var task task.Task

		if err := json.NewDecoder(res.Body).Decode(&task); err != nil {
			return fmt.Errorf("decode: %v", err)
		}

		if task.FinishedAt == nil {
			continue
		}

		log.Printf("task %s terminated on %v", task.ID, task.FinishedAt)

		return nil
	}
}
