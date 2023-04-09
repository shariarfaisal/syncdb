package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
)

type PGClient struct {
	*sql.DB
	URL    string
	Notify chan Notification
}

func NewPgConn(connStr string) PGClient {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	return PGClient{
		DB:     db,
		URL:    connStr,
		Notify: make(chan Notification),
	}
}

func (db *PGClient) createNotificationTriggers() {
	// Create a table
	if _, err := db.DB.Exec(`
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

	if _, err := db.DB.Exec(`
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
		fmt.Println(err)
	}

	// Create a trigger to call the trigger function on changes to the table
	if _, err := db.Exec("CREATE TRIGGER notification_changes AFTER INSERT OR UPDATE ON notification FOR EACH ROW EXECUTE FUNCTION notify_notification_changes()"); err != nil {
		println(err.Error())
	}
}

func (db *PGClient) ListenChanges() {
	defer db.DB.Close()
	db.createNotificationTriggers()

	// Set up a notification channel and subscribe to it
	if _, err := db.DB.Query("LISTEN notification_changes"); err != nil {
		fmt.Println(err)
	}

	// Start a goroutine to listen for notifications and print them
	go func(notify chan Notification) {
		notificationListener := pq.NewListener(db.URL, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Println(err)
			}
		})

		defer notificationListener.Close()

		if err := notificationListener.Listen("notification_changes"); err != nil {
			fmt.Println(err)
		}

		for n := range notificationListener.Notify {
			if n == nil {
				continue
			}

			var notification Notification
			if err := json.Unmarshal([]byte(n.Extra), &notification); err != nil {
				fmt.Println(err)
			}

			notification.Driver = "postgres"
			notify <- notification
		}

	}(db.Notify)
}
