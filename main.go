package main

import (
	"log"

	"github.com/Munchies-Engineering/syncdb/db"
	mainStream "github.com/Munchies-Engineering/syncdb/stream"
	"github.com/Munchies-Engineering/syncdb/util"
	"github.com/gin-gonic/gin"
)

func main() {

	config, err := util.LoadConfig("./")
	if err != nil {
		log.Fatal(err)
	}
	stream := mainStream.NewServer()

	mongoClient, err := db.NewMongoConn(config.MongoURL)
	if err != nil {
		log.Fatal(err)
	}
	go mongoClient.ListenChanges()

	go func(mongoNewDoc chan db.Notification) {
		for doc := range mongoNewDoc {
			stream.Message <- doc
		}
	}(mongoClient.Notify)

	pgClient := db.NewPgConn(config.PgURL)
	go pgClient.ListenChanges()

	go func(pgNewDoc chan db.Notification) {
		for doc := range pgNewDoc {
			stream.Message <- doc
		}
	}(pgClient.Notify)

	router := gin.Default()

	// Initialize new streaming server
	// Basic Authentication
	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": "admin",
	}))

	// Authorized client can stream the event
	// Add event-streaming headers
	authorized.GET("/stream", mainStream.HeadersMiddleware(), stream.ServeHTTP(), stream.Streaming())
	router.StaticFile("/", "./public/index.html")

	router.Run(":8085")
}
