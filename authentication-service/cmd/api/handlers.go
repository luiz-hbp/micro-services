package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type LogEntry struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusNotFound)
		return
	}
	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}
	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in as %s", user.Email),
		Data:    user,
	}
	logSucess := LogEntry{Name: "authentication", Data: fmt.Sprintf("%s logged at :%s", user.Email, time.Now())}
	err = app.log(logSucess)
	if err != nil {
		app.errorJSON(w, errors.New("failed to log"))
		return
	}
	app.writeJSON(w, http.StatusAccepted, payload)
	return
}

func (app *Config) log(a LogEntry) error {
	jsonData, err := json.Marshal(a)
	if err != nil {
		log.Println("failed to log")
		return err
	}

	req, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("failed to log")
		return err
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println("failed to log")
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Println("failed to log")
		return err
	}
	return nil
}
