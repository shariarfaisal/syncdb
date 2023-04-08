package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	mainStream "github.com/Munchies-Engineering/syncdb/stream"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// config stores all configuration ot the application
// The values are read by viper from a config file or environment variables

type Config struct {
	MongoURL string `mapstructure:"MONGO_URL"`
	PgURL    string `mapstructure:"PG_URL"`
}

// LoadConfig read configuration from file or env variables
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

type Document struct {
	OperationType     string                 `bson:"operationType"`
	FullDocument      map[string]interface{} `bson:"fullDocument"`
	DocumentKey       map[string]interface{} `bson:"documentKey"`
	UpdateDescription map[string]interface{} `bson:"updateDescription"`
	Ns                map[string]interface{} `bson:"ns"`
}

func mongoConn(notify chan Document) {
	config, err := LoadConfig("./")
	if err != nil {
		log.Fatal(err)
	}

	clientOptions := options.Client().ApplyURI(config.MongoURL)

	mongoClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// watch changes
	changeStream, err := mongoClient.Watch(context.Background(), mongo.Pipeline{})
	if err != nil {
		panic(err)
	}

	defer changeStream.Close(context.Background())

	for changeStream.Next(context.Background()) {
		var doc Document
		err := changeStream.Decode(&doc)
		if err != nil {
			panic(err)
		}
		notify <- doc
	}

	defer mongoClient.Disconnect(context.Background())
}

type Notification struct {
	TableName string                 `json:"table_name"`
	Operation string                 `json:"operation"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt string                 `json:"created_at"`
}

func listenPG(notify chan Notification) {
	config, err := LoadConfig("./")
	if err != nil {
		log.Fatal(err)
	}

	connStr := config.PgURL
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a table
	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS notification (
		id serial primary key,
		table_name varchar(100) not null,
		operation varchar(10) not null,
		data jsonb not null,
		created_at timestamp default now()
	);
	`); err != nil {
		fmt.Println(err)
	}

	if _, err := db.Exec(`
	CREATE OR REPLACE FUNCTION notify_changes() RETURNS trigger AS $$
		BEGIN
			IF (TG_OP = 'INSERT') THEN
				INSERT INTO notification (table_name, operation, data)
				VALUES (TG_TABLE_NAME, 'INSERT', row_to_json(NEW));
				RETURN NEW;
			ELSIF (TG_OP = 'UPDATE') THEN
				INSERT INTO notification (table_name, operation, data)
				VALUES (TG_TABLE_NAME, 'UPDATE', row_to_json(NEW));
				RETURN NEW;
			ELSIF (TG_OP = 'DELETE') THEN
				INSERT INTO notification (table_name, operation, data)
				VALUES (TG_TABLE_NAME, 'DELETE', row_to_json(OLD));
				RETURN OLD;
			END IF;
		END;
		$$ LANGUAGE plpgsql;
	`); err != nil {
		fmt.Println(err)
	}

	rows, err := db.Query(`
		SELECT format('CREATE TRIGGER notify_changes_trigger_%1$s
		AFTER INSERT OR UPDATE OR DELETE
		ON %1$s
		FOR EACH ROW
		EXECUTE PROCEDURE notify_changes();', table_name)
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name NOT IN ('notification')
	`)
	if err != nil {
		fmt.Println(err)
	}

	defer rows.Close()

	var triggerStmt string
	for rows.Next() {
		err := rows.Scan(&triggerStmt)
		if err != nil {
			panic(err)
		}

		query := string(triggerStmt)
		go func(query string) {
			_, err := db.Exec(query)

			if err != nil {
				fmt.Println(err)
			}
		}(query)
	}

	// Create a trigger function to send notifications on changes to the table
	if _, err := db.Exec(`
		CREATE OR REPLACE FUNCTION notify_notification_changes() RETURNS TRIGGER AS $$
			DECLARE
				notification_payload JSON;
			BEGIN
				notification_payload = row_to_json(NEW);
				PERFORM pg_notify('notification_changes', notification_payload::text);
				RETURN NEW;
			END;
		$$ LANGUAGE plpgsql;
	`); err != nil {
		log.Fatal(err)
	}

	// Create a trigger to call the trigger function on changes to the table
	if _, err := db.Exec("CREATE TRIGGER notification_changes AFTER INSERT OR UPDATE ON notification FOR EACH ROW EXECUTE FUNCTION notify_notification_changes()"); err != nil {
		println(err.Error())
	}

	// Set up a notification channel and subscribe to it
	if _, err := db.Exec("LISTEN notification_changes"); err != nil {
		log.Fatal(err)
	}

	// Start a goroutine to listen for notifications and print them
	go func(notify chan Notification) {
		notificationListener := pq.NewListener(config.PgURL, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Println(err)
			}
		})
		defer notificationListener.Close()

		if err := notificationListener.Listen("notification_changes"); err != nil {
			log.Fatal(err)
		}

		for n := range notificationListener.Notify {
			if n == nil {
				continue
			}

			var notification Notification
			if err := json.Unmarshal([]byte(n.Extra), &notification); err != nil {
				log.Fatal(err)
			}

			notify <- notification
		}

	}(notify)
}

func main() {

	stream := mainStream.NewServer()

	mongoNewDoc := make(chan Document)
	go mongoConn(mongoNewDoc)

	go func(mongoNewDoc chan Document) {
		for doc := range mongoNewDoc {
			fmt.Println("Mongodb", doc)
			stream.Message <- doc
		}
	}(mongoNewDoc)

	pgNewDoc := make(chan Notification)
	go listenPG(pgNewDoc)

	go func(pgNewDoc chan Notification) {
		for doc := range pgNewDoc {
			fmt.Println("Postgres", doc.Data)
			stream.Message <- doc.Data
		}
	}(pgNewDoc)

	router := gin.Default()

	// Initialize new streaming server

	// Basic Authentication
	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": "admin",
	}))

	// Authorized client can stream the event
	// Add event-streaming headers
	authorized.GET("/stream", mainStream.HeadersMiddleware(), stream.ServeHTTP(), func(c *gin.Context) {
		v, ok := c.Get("clientChan")
		if !ok {
			return
		}
		clientChan, ok := v.(mainStream.ClientChan)
		if !ok {
			return
		}
		c.Stream(func(w io.Writer) bool {
			// Stream message to client from message channel
			if msg, ok := <-clientChan; ok {
				c.SSEvent("message", msg)
				return true
			}
			return false
		})
	})

	// Parse Static files
	router.StaticFile("/", "./index.html")

	router.Run(":8085")
}
