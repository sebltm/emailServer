/*
Created by 660046669 on 12/02/2020
This bluebook agent handles providing information about server addresses based
on the SLD name provided
*/

package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Server registered with the bluebook
var servers map[string]Server

// Server struct representing a server, with name and address
type Server struct {
	Name    string
	Address string
}

func main() {
	servers = make(map[string]Server)

	handleRequests()
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/bluebook/{email}", FindServer).Methods("GET")
	router.HandleFunc("/bluebook/register", AddServer).Methods("POST")

	log.Fatal(http.ListenAndServe(":8888", router))
}

// FindServer returns the JSON object corresponding to the server requested
func FindServer(w http.ResponseWriter, r *http.Request) {

	// Extract the domain from the email
	vars := mux.Vars(r)
	email := vars["email"]
	emailSplit := strings.Split(email, "@")
	_, domain := emailSplit[0], emailSplit[1]

	log.Print("New request: " + domain)

	server, ok := servers[domain]

	// Log whether the requests matches or not
	if ok {
		log.Print("Request matches: " + server.Name + " @ " + server.Address)
	} else {

		// If there's no match, send a 404
		log.Print("No matches for request")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Format as JSON and send the response
	if serverByte, err := json.Marshal(server); err == nil {
		w.WriteHeader(http.StatusOK)
		w.Write(serverByte)
	} else {
		log.Print(err.Error())
		// Oops, JSON marshal problem...
		w.WriteHeader(http.StatusInternalServerError)
	}

}

// AddServer adds a new server to the list of servers, with the name and
// corresponding address of the server
func AddServer(w http.ResponseWriter, r *http.Request) {
	var newServer Server

	// Read the body
	body, err := ioutil.ReadAll(r.Body)
	log.Print(string(body))

	if err != nil {
		log.Print(err.Error())
		// Error while reading the body...
		w.WriteHeader(http.StatusInternalServerError)
	}

	if err := json.Unmarshal(body, &newServer); err == nil {
		w.WriteHeader(http.StatusCreated)
		servers[newServer.Name] = newServer

		log.Println("Registered " + newServer.Name)
	} else {
		// If we can't unmarshal the json, we're assuming the client made a mistake
		// in formatting their request
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
	}

}
