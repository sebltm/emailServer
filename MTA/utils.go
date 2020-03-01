package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"
)

// EMail struct representing an email
type EMail struct {
	UUID     uuid.UUID
	Sender   string
	Receiver string
	Object   string
	Message  string
}

// Server struct representing an MTA server
type Server struct {
	Name    string
	Address string
}

// MSA struct representing an MSA client
type MSA struct {
	Name    string
	Address string
}

// Folder struct representing a folder of email (inbox or outbox)
type Folder struct {
	Emails []EMail
}

// MSA clients registered with this MTA server
var msa map[string]MSA

// This MTA server
var self Server

// GetOutboundIP allows us to get the outbound IP of this agent, this allows us
// not to have to indicate it at startup as command line argument
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

//CreateDirIfNotExist creates a directory and subdirectory-ies if they don't
//exist, with read-write permissions
func CreateDirIfNotExist(dir string) {

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err.Error())
		}
	}
}

// Register this MTA with the bluebook service
func register(self Server) {
	selfJSON, err := json.Marshal(self)

	// if we couldn't register, panic!
	if err != nil {
		panic(err.Error())
	}

	resp, err := http.Post("http://192.168.1.3:8888/bluebook/register",
		"application/json", bytes.NewReader(selfJSON))

	// if we couldn't register, panic !
	if resp.StatusCode > 299 || err != nil {
		log.Print("Status code : " + resp.Status)
		log.Print(err.Error())
		panic("The bluebook service is unavailabe, or there was a" +
			" problem while sending the request")
	}

	log.Print("Registered!")
}

func deleteEmail(address string, email EMail) {
	client := &http.Client{}

	// send a request to delete the email
	deleteReq, err := http.NewRequest("DELETE",
		address+"email/outbox/"+email.UUID.String(), nil)

	// if there was a problem while creating the request, log the error and exit
	if err != nil {
		log.Print(err.Error())
		return
	}

	_, err = client.Do(deleteReq)

	if err != nil {
		log.Fatal(err.Error())
		return
	}
}
