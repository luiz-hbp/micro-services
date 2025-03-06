package main

import (
	"auth/data"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const webPort = "80"

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println("Starting authentication service")
	conn := connectToDB()
	if conn == nil {
		log.Panic("Couldn`t connect to Postgres")
	}
	app := Config{
		DB:     conn,
		Models: data.New(conn),
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func connectToDB() *sql.DB {
	var countErr int64
	dsn := os.Getenv("DSN")
	for {
		connection, err := openDB(dsn)
		if err != nil {
			countErr++
			log.Println("Postgres not ready yet...")
		} else {
			log.Println("Connected to Postgres!")
			return connection
		}
		if countErr > 10 {
			log.Println(err)
			return nil
		}
		log.Println("Retrying in two seconds...")
		time.Sleep(2 * time.Second)
		continue
	}
}
