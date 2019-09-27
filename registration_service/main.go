package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

type DbConfig struct {
	DbDriver string `envconfig:"DB_DRIVER" default:"postgres"`
	DbDSN    string `envconfig:"DB_DSN" required:"true"`
}

type AmqpConfig struct {
	AmqpDSN string `envconfig:"AMQP_DSN" required:"true"`
}

type config struct {
	DbConfig
	AmqpConfig
	ServerAddr string `envconfig:"SERVER_ADDR" required:"true"`
}

func main() {
	var conf config
	failOnError(envconfig.Process("reg_service", &conf), "failed to init config")

	// PostgresSQL
	pgSQL, err := sqlx.Connect(conf.DbDriver, conf.DbDSN)
	failOnError(err, "failed to connect to db")
	defer failOnClose(pgSQL, "close connect to db error")

	// RabbitMQ
	conn, err := amqp.Dial(conf.AmqpDSN)
	failOnError(err, "failed to connect to RabbitMQ")
	defer failOnClose(conn, "failed to close RMQ connection")

	rmqCh, err := conn.Channel()
	failOnError(err, "failed to open RMQ channel")
	defer failOnClose(rmqCh, "failed to close RMQ channel")

	// HTTP server
	handler := registrationHandler{
		db:        pgSQL,
		publisher: rmqCh,
	}
	s := &http.Server{
		Addr:           conf.ServerAddr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 Mb
	}
	log.Printf("Listen at %s", conf.ServerAddr)
	failOnError(s.ListenAndServe(), "starting server error")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func failOnClose(closer io.Closer, msg string) func() {
	return func() {
		failOnError(closer.Close(), msg)
	}
}
