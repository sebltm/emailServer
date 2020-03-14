package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// INBOX is global variable to reference INBOX
const INBOX = 0

// OUTBOX is a global variable to reference OUTBOX
const OUTBOX = 1

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

// This MSA
var self Server

// register itself as an active MSA client to the MTA server handling the
// emailSplit for this particular SLD domain. All errors in this function are
// handled as fatal, since otherwise the MSA client won't be able to register
// with its MTA server
func register(self Server) {
	registered := false

	for !registered {
		// Bluebook IP has to be hardcoded... think of it like a DNS server !
		// Request the IP address of the SLD server for this client
		responseBlueBook, err := http.Get("http://192.168.1.3:8888/bluebook/" +
			self.Name)

		// If there was a failure during registration, maybe the Blue Book server
		// doesn't exist (404) or is unresponsive (5xx), then don't let the MSA
		// client startup. log and loop again
		if err != nil {
			log.Print(err.Error())
			time.Sleep(2 * time.Second)
			continue
		} else if responseBlueBook.StatusCode > 299 {
			log.Print("The bluebook service is unavailabe, or there was a" +
				" problem while sending the request : " + responseBlueBook.Status)
			time.Sleep(2 * time.Second)
			continue
		}

		bodyBlueBook, err := ioutil.ReadAll(responseBlueBook.Body)

		if err != nil {
			// again, if we couldn't read the response from the bluebook, no point
			// in starting up, so loop again
			log.Print(err.Error())
			time.Sleep(2 * time.Second)
			continue
		}

		var MTAServ Server
		err = json.Unmarshal(bodyBlueBook, &MTAServ)

		if err != nil {
			// again, if we couldn't read the response from the bluebook, no point
			// in starting up, so loop again
			log.Print(err.Error())
			time.Sleep(2 * time.Second)
			continue
		}

		selfJSON, err := json.Marshal(self)

		if err != nil {
			// If we can't create json string to describe this client, loop again
			log.Print(err.Error())
			time.Sleep(2 * time.Second)
			continue
		}

		log.Print("Registering with:", MTAServ.Address+"email/server/register")

		respMTA, err := http.Post(MTAServ.Address+"email/server/register", "application/json",
			bytes.NewReader(selfJSON))

		// If there was a failure during registration, maybe the MTA server
		// doesn't exist (404) or is unresponsive (5xx), then don't let the MSA
		// client startup. log and loop again
		if err != nil {
			log.Print(err.Error())
			time.Sleep(2 * time.Second)
			continue
		} else if responseBlueBook.StatusCode > 299 {
			log.Print("The bluebook service is unavailabe, or there was a" +
				" problem while sending the request : " + respMTA.Status)
			time.Sleep(2 * time.Second)
			continue
		}

		// Finally registered ! Log and break out of the loop
		registered = true

		log.Print("Registered!")
	}
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
