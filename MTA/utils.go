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
	UUID    uuid.UUID
	From    string
	To      string
	Subject string
	Body    string
}

// Server struct representing an MTA server
type Server struct {
	Name    string
	Address string
}

// Folder struct representing a folder of email (inbox or outbox)
type Folder struct {
	Emails []EMail
}

// MSA clients registered with this MTA server
var msa map[string]Server

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
	registered := false

	// Keep trying to register until Blue Book is available
	for !registered {
		selfJSON, err := json.Marshal(self)

		// if we couldn't register, panic!
		if err != nil {
			panic(err.Error())
		}

		resp, err := http.Post("http://192.168.1.3:8888/bluebook/register",
			"application/json", bytes.NewReader(selfJSON))

		// if we couldn't register, log and loop again
		if err != nil {
			log.Print(err.Error())
			continue
		} else if resp.StatusCode > 299 {
			log.Print("The bluebook service is unavailabe, or there was a" +
				" problem while sending the request : " + resp.Status)
			continue
		}

		// Finally registered ! Log and break out of the loop
		registered = true
		log.Print("Registered!")
	}
}
