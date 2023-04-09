package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDocument struct {
	OperationType     string                 `bson:"operationType"`
	FullDocument      map[string]interface{} `bson:"fullDocument"`
	DocumentKey       map[string]interface{} `bson:"documentKey"`
	UpdateDescription map[string]interface{} `bson:"updateDescription"`
	Ns                map[string]interface{} `bson:"ns"`
}

type MongoConnection struct {
	Client *mongo.Client
	URL    string
	Notify chan Notification
}

func NewMongoConn(connStr string) (MongoConnection, error) {

	clientOptions := options.Client().ApplyURI(connStr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return MongoConnection{}, err
	}

	return MongoConnection{
		URL:    connStr,
		Notify: make(chan Notification),
		Client: client,
	}, nil
}

func (conn *MongoConnection) ListenChanges() {

	changeStream, err := conn.Client.Watch(context.Background(), mongo.Pipeline{})
	if err != nil {
		fmt.Println("42: ", err)
		return
	}

	defer changeStream.Close(context.Background())

	for changeStream.Next(context.Background()) {
		var doc MongoDocument
		err := changeStream.Decode(&doc)
		if err != nil {
			fmt.Println(err)
		}

		notification := Notification{
			Driver:    "mongo",
			TableName: doc.Ns["coll"].(string),
			Operation: doc.OperationType,
		}

		if doc.OperationType == "insert" {
			notification.Data = doc.FullDocument
		} else if doc.OperationType == "update" {
			notification.Data = doc.UpdateDescription
			notification.Data["documentKey"] = doc.DocumentKey
		} else if doc.OperationType == "delete" {
			notification.Data = doc.DocumentKey
		}

		conn.Notify <- notification
	}

	if err := changeStream.Err(); err != nil {
		fmt.Println(err)
	}

}
