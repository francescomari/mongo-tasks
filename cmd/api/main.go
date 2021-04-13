package main

import (
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
	"github.com/gorilla/mux"
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

	createTaskHandler := createTaskHandler{
		database: &database,
	}

	readTaskHandler := readTaskHandler{
		database: &database,
	}

	router := mux.NewRouter()

	router.
		Methods(http.MethodPost).
		Path("/tasks").
		Handler(&createTaskHandler)

	router.
		Methods(http.MethodGet).
		Path("/tasks/{taskId}").
		Handler(&readTaskHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalCh

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("error: shutdown: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
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

type createTaskHandler struct {
	database *task.Database
}

func (h *createTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	createCtx, createCancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer createCancel()

	id, err := h.database.CreateTask(createCtx, &task.Task{
		ID:        uuid.NewString(),
		CreatedAt: time.Now(),
		Data:      data,
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	readCtx, readCancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer readCancel()

	task, err := h.database.ReadTask(readCtx, id)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	if err := json.NewEncoder(w).Encode(task); err != nil {
		log.Printf("encode: %v", err)
	}
}

type readTaskHandler struct {
	database *task.Database
}

func (h *readTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	task, err := h.database.ReadTask(ctx, mux.Vars(r)["taskId"])
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")

	if err := json.NewEncoder(w).Encode(task); err != nil {
		log.Printf("encode: %v", err)
	}
}
