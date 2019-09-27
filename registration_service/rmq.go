package main

import (
	"io"

	"github.com/streadway/amqp"
)

type Publisher interface {
	io.Closer
	amqp.Acknowledger
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}
