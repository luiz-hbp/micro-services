package main

import (
	"broker/event"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type RequestPayload struct {
	Action string       `json:"action"`
	Auth   AuthPayload  `json:"auth,omitempty"`
	Log    LogPayload   `json:"log,omitempty"`
	Email  EmailPayload `json:"email,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type EmailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	app.writeJSON(w, 200, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	err := json.NewDecoder(r.Body).Decode(&requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
		return
	case "log":
		app.logEventViaRabbit(w, requestPayload.Log)
		return
	case "email":
		app.mail(w, requestPayload.Email)
		return
	default:
		app.errorJSON(w, errors.New("action unavailable"), http.StatusBadRequest)
		return
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	jsonData, err := json.Marshal(a)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	req, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("unvalid credentials"), http.StatusUnauthorized)
		return
	} else if res.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error contacting auth service"), http.StatusUnauthorized)
		return
	}

	var authResponse jsonResponse
	err = json.NewDecoder(res.Body).Decode(&authResponse)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if authResponse.Error {
		app.errorJSON(w, errors.New("unathorized by the auth service"), http.StatusUnauthorized)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = authResponse.Data
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) log(w http.ResponseWriter, a LogPayload) {
	jsonData, err := json.Marshal(a)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	req, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error contacting log service"), http.StatusInternalServerError)
		return
	}

	var logResponse jsonResponse
	err = json.NewDecoder(res.Body).Decode(&logResponse)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if logResponse.Error {
		app.errorJSON(w, errors.New("failed to communicate with log service"), http.StatusInternalServerError)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged!"
	payload.Data = logResponse.Data
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) mail(w http.ResponseWriter, a EmailPayload) {
	jsonData, err := json.Marshal(a)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	req, err := http.NewRequest("POST", "http://mail-service/send", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error contacting email service"), http.StatusInternalServerError)
		return
	}

	var mailResponse jsonResponse
	err = json.NewDecoder(res.Body).Decode(&mailResponse)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if mailResponse.Error {
		app.errorJSON(w, errors.New("failed to communicate with email service"), http.StatusInternalServerError)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Email sent!"
	payload.Data = mailResponse.Data
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via rabbitMQ"

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name string, msg string) error {
	emitter, err := event.NewEventEmitter(app.rabbit)
	if err != nil {
		return err
	}
	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	json, err := json.Marshal(&payload)
	if err != nil {
		return err
	}

	err = emitter.Push(string(json), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}
