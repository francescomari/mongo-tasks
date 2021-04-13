package task

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Task is a task that will be submitted via the API and executed asynchronously
// by a worker.
type Task struct {
	ID         string                 `bson:"_id" json:"id"`
	Data       map[string]interface{} `bson:"data" json:"data"`
	CreatedAt  time.Time              `bson:"createdAt" json:"createdAt"`
	StartedAt  *time.Time             `bson:"startedAt" json:"startedAt,omitempty"`
	FinishedAt *time.Time             `bson:"finishedAt" json:"finishedAt,omitempty"`
}

// Database access task data from a MongoDB database.
type Database struct {
	URI      string
	Database string
	client   *mongo.Client
}

// Connect connects to the database. Returns an error if the connection can't be
// established.
func (db *Database) Connect(ctx context.Context) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(db.URI))
	if err != nil {
		return fmt.Errorf("connect: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("ping: %v", err)
	}

	db.client = client

	return nil
}

// Disconnect disconnects from the database. Returns an error if some resources
// couldn't be returned before the context expires.
func (db *Database) Disconnect(ctx context.Context) error {
	if err := db.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect: %v", err)
	}

	db.client = nil

	return nil
}

func (db *Database) tasks() *mongo.Collection {
	return db.client.Database(db.Database).Collection("tasks")
}

// CreateTask creates a new task. The ID, data and created time must be
// specified. The started and finished time must not. Returns an error if the
// task is invalid or if an error occurs when creating the task.
func (db *Database) CreateTask(ctx context.Context, task *Task) (string, error) {
	if task.ID == "" {
		return "", fmt.Errorf("field ID is empty")
	}
	if task.Data == nil {
		return "", fmt.Errorf("field Data is nil")
	}
	if task.CreatedAt.IsZero() {
		return "", fmt.Errorf("field CreatedAt is zero")
	}
	if task.StartedAt != nil {
		return "", fmt.Errorf("field StartedAt is not nil")
	}
	if task.FinishedAt != nil {
		return "", fmt.Errorf("field FinishedAt is not nil")
	}

	result, err := db.tasks().InsertOne(ctx, task)
	if err != nil {
		return "", fmt.Errorf("insert one: %v", err)
	}

	return result.InsertedID.(string), nil
}

// ReadTask reads a task given its ID. Returns a nil task if the task can't be
// found. Returns an error if the arguments are not valid or if an error occurs
// when reading the task.
func (db *Database) ReadTask(ctx context.Context, id string) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("id is empty")
	}

	result := db.tasks().FindOne(ctx, bson.M{"_id": id})
	if result.Err() == mongo.ErrNoDocuments {
		return nil, nil
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("find one: %v", result.Err())
	}

	var task Task

	if err := result.Decode(&task); err != nil {
		return nil, fmt.Errorf("decode: %v", err)
	}

	return &task, nil
}

// StartTask picks a runnable task from the database and updates its start time.
// Returns a nil task if no runnable tasks are found. Returns an error if an
// error occurs when reading the task from the database.
func (db *Database) StartTask(ctx context.Context) (*Task, error) {

	// Sort the documents from the oldest to the newest. Only the first will be
	// updated.

	sort := bson.M{
		"createdAt": 1,
	}

	// Filter only the documents that haven't been picked up by the worker, i.e.
	// the ones with a nil start date.

	filter := bson.M{
		"startedAt": nil,
	}

	// Update the found document and assign it a start date, so it won't be
	// picked up by any other worker.

	update := bson.M{
		"$set": bson.M{
			"startedAt": time.Now(),
		},
	}

	// Return the document as modified after the update.

	returnDocument := options.After

	options := options.FindOneAndUpdateOptions{
		Sort:           sort,
		ReturnDocument: &returnDocument,
	}

	result := db.tasks().FindOneAndUpdate(ctx, filter, update, &options)
	if result.Err() == mongo.ErrNoDocuments {
		return nil, nil
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("find one and update: %v", result.Err())
	}

	var task Task

	if err := result.Decode(&task); err != nil {
		return nil, fmt.Errorf("decode: %v", err)
	}

	return &task, nil
}

// FinishTask sets the finished time of a task. Returns an error if the
// arguments are invalid or if an error occurs when updating the task in the
// database.
func (db *Database) FinishTask(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	filter := bson.M{
		"_id": id,
	}

	update := bson.M{
		"$set": bson.M{
			"finishedAt": time.Now(),
		},
	}

	result := db.tasks().FindOneAndUpdate(ctx, filter, update)
	if result.Err() != nil {
		return fmt.Errorf("find one and update: %v", result.Err())
	}

	return nil
}
