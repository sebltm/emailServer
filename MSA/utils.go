package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"
)

// INBOX is global variable to reference INBOX
const INBOX = 0

// OUTBOX is a global variable to reference OUTBOX
const OUTBOX = 1

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

// This MSA
var self MSA

// register itself as an active MSA client to the MTA server handling the
// emailSplit for this particular SLD domain. All errors in this function are
// handled as fatal, since otherwise the MSA client won't be able to register
// with its MTA server
func register(self MSA) {

	// Bluebook IP has to be hardcoded... think of it like a DNS server !
	// Request the IP address of the SLD server for this client
	responseBlueBook, err := http.Get("http://192.168.1.3:8888/bluebook/" +
		self.Name)

	if err != nil {
		// if the Bluebook either doesn't exist (404) or is unresponsive (5xx),
		// don't let the MSA client startup
		panic(err.Error())
	}

	bodyBlueBook, err := ioutil.ReadAll(responseBlueBook.Body)

	if err != nil {
		// again, if we couldn't read the response from the bluebook, no point
		// in starting up, so exit with error
		panic(err.Error())
	}

	var MTAServ Server
	err = json.Unmarshal(bodyBlueBook, &MTAServ)

	if err != nil {
		// again, if we couldn't read the response from the bluebook, no point
		// in starting up, so exit with error
		panic(err.Error())
	}

	selfJSON, err := json.Marshal(self)

	if err != nil {
		// If we can't create json string to describe this client, exit
		panic(err.Error())
	}

	log.Print("Registering with:", MTAServ.Address+"email/server/register")

	_, err = http.Post(MTAServ.Address+"email/server/register", "application/json",
		bytes.NewReader(selfJSON))

	if err != nil {
		// If there was a failure during registration, maybe the MTA server
		// doesn't exist (404) or is unresponsive (5xx), then don't let the MSA
		// client startup
		panic(err.Error())
	}

	log.Print("Registered!")
}

//CreateDirIfNotExist creates a directory and subdirectory-ies if they don't
//exist. with read-write permissions
func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err.Error())
		}
	}
}

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
