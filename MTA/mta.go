/*
Created by 660046669 on 12/02/2020
mta.go is the "Mail Transfer Agent"
This agent handles receiving mail from other MTAs, and dispatches the emails
to the correct MSA
This agent also handles periodic (15s) scanning of all the MSA outboxes,
and sends the emails to the correct MSA
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	msa = make(map[string]Server)

	if len(os.Args) < 2 {
		fmt.Println("To run the MTA service, please provide a domain name.")
		fmt.Println("e.g 'go run mta.go domain.com'")
		fmt.Println("or with Docker: 'docker run MSA-image domain.com'")
		fmt.Println()
		os.Exit(1)
	}

	self.Name = os.Args[1]

	// Register with the Bluebook service
	self.Address = "http://" + GetOutboundIP() + ":8888/"

	// Register in the background -> loosely coupled! we want the MTA to keep
	// serving requests independently of whether the Blue Book works or not
	go register(self)

	// Start a gorountine to handle requests in background
	go handleRequests()

	// Periodically scan outboxes registered with this MTA to send emails
	done := make(chan bool)
	ticker := time.NewTicker(15 * time.Second)

	for {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-ticker.C:
			MTAScanAndSend()
		}
	}
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/email/server", MTAServe).Methods("POST")
	router.HandleFunc("/email/server/register", AddMSA).Methods("POST")

	log.Fatal(http.ListenAndServe(":8888", router))
}

// AddMSA adds a new MSA to the list of clients, with the name and
// corresponding address of the client
func AddMSA(w http.ResponseWriter, r *http.Request) {
	var newMSA Server

	body, err := ioutil.ReadAll(r.Body)

	// If we can't read the body, exit with error
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(body, &newMSA); err == nil {
		w.WriteHeader(http.StatusCreated)
		msa[newMSA.Name] = newMSA
		log.Println("Registered a new MSA client : "+newMSA.Name, newMSA.Address)
	} else {
		// If we can't unmarshal the json, we're assuming the client made a mistake
		// in formatting their request
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
	}

}

// MTAServe handles forwarding the email sent by other MTAs to this MTA, and
// dispatching to the right MSAs
func MTAServe(w http.ResponseWriter, r *http.Request) {
	var email EMail

	body, err := ioutil.ReadAll(r.Body)

	// If we can't read the body, exit with error
	if err != nil {
		log.Println("Could not read body " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &email)

	// If we can't unmarshal the json, we're assuming the client made a mistake
	// in formatting their request
	if err != nil {
		log.Println("Could not unmarshal email " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Find the address of the MSA client
	recipient := msa[email.To]

	// Re-marshal the email
	emailJSON, err := json.Marshal(email)

	// Forward to the MSA
	resp, err := http.Post(recipient.Address+"email/outbox", "application/json",
		bytes.NewReader(emailJSON))

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 && err == nil {
		w.WriteHeader(http.StatusOK)
		log.Printf("Delivered email %s to %s", email.Subject, email.To)
	} else if (resp.StatusCode < 200 || resp.StatusCode > 299) && err != nil {
		// forward the error to the MTA
		w.WriteHeader(resp.StatusCode)
		log.Print("Couldn't dispatch : " + resp.Status)
	} else if err != nil {
		// error occured while dispatching, tell the MTA
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
	}

}

// MTAScanAndSend scans all the outboxes on the server and sends all the emails
func MTAScanAndSend() {
	var destServer Server

	var folder Folder

	// iterate over all the MSAs registered with this MTA
	for _, msaObj := range msa {
		log.Print("Scan and serve: " + msaObj.Address + " (" + msaObj.Name + ")")
		responseMSA, err := http.Get(msaObj.Address + "email/outbox")

		// couldn't get the mail from this outbox... log and move on to the next
		if err != nil {
			log.Print(err.Error())
			continue
		}

		bodyMSA, err := ioutil.ReadAll(responseMSA.Body)

		// couldn't get the mail from this outbox... log and move on to the next
		if err != nil {
			log.Print(err.Error())
			continue
		}

		err = json.Unmarshal(bodyMSA, &folder)

		// could not unmarshal the mail from this outbox... assuming that the MSA
		// handled all emails being well formed, this is an error of the unmarshal
		// so log and move on, it should be read at the next "scan and serve"
		if err != nil {
			log.Print(err.Error())
			continue
		}

		log.Printf("Found %d emails !\n", len(folder.Emails))

		// Process each email in the folder
		for _, email := range folder.Emails {
			log.Println("Sending " + email.Subject)

			// Ask the bluebook who this email should go to
			blueBookRequest := "http://192.168.1.3:8888/bluebook/" + email.To
			blueBookResponse, err := http.Get(blueBookRequest)

			address := msa[email.From].Address

			if (blueBookResponse.StatusCode <= 400 ||
				blueBookResponse.StatusCode >= 499) && err != nil {
				// problem with the email we're sending so delete it
				deleteEmail(address, email)
				log.Print("Problem with email. Email has been deleted")
				continue
			} else if (blueBookResponse.StatusCode <= 500 ||
				blueBookResponse.StatusCode >= 599) && err != nil {
				// error occured while contacting BlueBook, retry later
				log.Println("Error while sending the request to the BlueBook ",
					blueBookResponse.Status)
				continue
			} else if err != nil {
				log.Println(err.Error())
				continue
			}

			// Read the reponse and unmarshal into a Server struct
			blueBookBody, err := ioutil.ReadAll(blueBookResponse.Body)

			if err != nil {
				log.Println("Could not read BlueBook response " + err.Error())
				return
			}

			err = json.Unmarshal(blueBookBody, &destServer)

			if err != nil {
				log.Println("Could not unmarshal BlueBook response " + err.Error())
				return
			}

			// format the  EMail struct into a JSON object, reading for sending
			emailJSON, err := json.Marshal(email)

			if err != nil {
				log.Println("Could not marshal email " + err.Error())
				return
			}

			serverPath := destServer.Address + "email" + "/server"

			// Finally, POST the email to the correct MTA !
			req, err := http.NewRequest("POST", serverPath,
				bytes.NewReader(emailJSON))

			if err != nil {
				// error while creating the request, try again later
				log.Println(err.Error())
				return
			}

			client := &http.Client{}
			resp, err := client.Do(req)

			// Here we deal with the reponse from the desintation
			// If it is unavailable, or there was an error with the request itself,
			// leave the email in the outbox and deal with it later. For any other
			// error, or if everything went okay, delete the email from the MSA's
			// outbox
			if (resp.StatusCode >= 500 && resp.StatusCode <= 599) && err == nil {
				// the MTA is currently unavailable, exit here and come back later
				log.Print("Destination MTA unavailable " + resp.Status +
					", retry later")
				return
			} else if err == nil {
				deleteEmail(address, email)
			} else {
				log.Print(err.Error())
			}
		}
	}
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

	resp, err := client.Do(deleteReq)

	if (resp.StatusCode > 299) && err == nil {
		// the MTA is currently unavailable, exit here and come back later
		log.Print("Could not delete email " + resp.Status + ", retry later")
		return
	} else if err != nil {
		log.Print(err.Error())
	}

}
