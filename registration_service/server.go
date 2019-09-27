package main

import (
	"encoding/json"
	"net/http"

	"github.com/streadway/amqp"
)

var registrationsExchangeName = "UserRegistrations" // Тоже по-хорошему необходимо выностить в конфиг

type registrationHandler struct {
	db        NamedExecer
	publisher Publisher
}

// Я специально не добавлял информации об ошибках, чтобы если у вас что-то не работает, вы копались сами :)
func (h registrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		h.ping(w, r)
		return

	case "/api/v1/registration":
		h.handleRegistration(w, r)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (h registrationHandler) ping(w http.ResponseWriter, r *http.Request) {
	if _, err := h.db.Exec("SELECT 1"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if h.publisher.Publish(registrationsExchangeName, "", false, false, amqp.Publishing{
		ContentType: "plain/text",
		Body:        []byte("HealthCheck"),
	}) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

func (h registrationHandler) handleRegistration(w http.ResponseWriter, r *http.Request) {
	user := user{}
	if json.NewDecoder(r.Body).Decode(&user) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := h.db.NamedQuery(`
		INSERT INTO users (first_name, email, age)
		VALUES (:first_name, :email, :age)
	`, user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userData, _ := json.Marshal(user)
	if h.publisher.Publish(registrationsExchangeName, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        userData,
	}) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
