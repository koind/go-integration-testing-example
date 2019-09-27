package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/streadway/amqp"
)

var (
	// Тоже по-хорошему необходимо выностить в конфиг
	queueName                 = "ToNotificationService"
	registrationsExchangeName = "UserRegistrations"
	notificationsExchangeName = "UserNotifications"
)

type AmqpConfig struct {
	AmqpDSN string `envconfig:"AMQP_DSN" required:"true"`
}

type config struct {
	AmqpConfig
}

type user struct {
	FirstName string
	Email     string
	Age       uint8
}

func main() {
	var conf config
	failOnError(envconfig.Process("notify_service", &conf), "failed to init config")

	// RabbitMQ
	conn, err := amqp.Dial(conf.AmqpDSN)
	failOnError(err, "failed to connect to RabbitMQ")
	defer failOnClose(conn, "failed to close RMQ connection")

	rmqCh, err := conn.Channel()
	failOnError(err, "failed to open RMQ channel")
	defer failOnClose(rmqCh, "failed to close RMQ channel")

	// Consume
	_, err = rmqCh.QueueDeclare(queueName, true, true, true, false, nil)
	failOnError(err, fmt.Sprintf("failed to create %s queue", queueName))

	err = rmqCh.QueueBind(queueName, "", registrationsExchangeName, false, nil)
	failOnError(err, "failed to bind notifications queue to exchange")

	users, err := rmqCh.Consume(queueName, "", true, false, false, false, nil)
	failOnError(err, "failed to register consumer")

	go func() {
		for userData := range users {
			if userData.ContentType != "application/json" {
				continue
			}

			var user user
			err := json.Unmarshal(userData.Body, &user)
			if err != nil {
				log.Printf("invalid user %s: %v", userData.Body, user)
			}

			// Эмулируем работу по отправке уведомлений пользователю
			time.Sleep(2 * time.Second)

			err = rmqCh.Publish(notificationsExchangeName, "email", false, false, amqp.Publishing{
				ContentType: "plain/text",
				Body:        []byte(user.Email),
			})
			if err != nil {
				log.Printf("invalid user %s: %v", userData.Body, user)
			}
		}
	}()

	log.Printf("Wait for new user registrations...")
	forever := make(chan struct{})
	<-forever
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
