package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Status struct {
	Status      string `json:"status"`
	LastUpdated string `json:"last_updated"`
}

type Message struct {
	Status    string `json:"status"`
	Body      string `json:"body"`
	CreatedOn string `json:"created_on"`
}

func getMessage() (Message, error) {
	m := Message{}

	resp, err := http.Get("https://status.github.com/api/last-message.json")
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Message{}, err
	}

	err = json.Unmarshal(content, &m)
	if err != nil {
		return Message{}, err
	}

	return m, nil
}

func getStatus() (Status, error) {
	s := Status{}

	resp, err := http.Get("https://status.github.com/api/status.json")
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Status{}, err
	}

	err = json.Unmarshal(content, &s)
	if err != nil {
		return Status{}, err
	}

	return s, nil
}
