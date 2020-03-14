/*
Created by 660046669 on 12/02/2020
msa.go is the "Mail Service Agent"
This agent handles the mail for a user, in the format of an email address
A user can display his entire mailbox, read one email, delete an email,
or send an email
Be aware ! There is no authentication, so any user can read any other user's
messages... but this is out of the scope of this coursework ?
This agent permanently stores messages in directories and files
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("To run the MSA service, please provide a user name.")
		fmt.Println("e.g 'go run mta.go user@domain.com'")
		fmt.Println("or with Docker: 'docker run MSA-image user@domain.com'")
		fmt.Println()
		os.Exit(1)
	}

	CreateDirIfNotExist("Outbox")
	CreateDirIfNotExist("Inbox")

	self.Name = os.Args[1]

	// Register with the correct MTA
	self.Address = "http://" + GetOutboundIP() + ":8888/"
	println("New MSA Client : "+self.Name, self.Address)

	// Register in the background -> loosely coupled! we want the MSA to keep
	// serving requests independently of whether the MSA and Blue Book work or not
	go register(self)

	handleRequests()
}

func handleRequests() {

	router := mux.NewRouter().StrictSlash(true)

	// MTA 'service' methods
	router.HandleFunc("/email/outbox", MSAReceive).Methods("POST")
	router.HandleFunc("/email/outbox", MSAReadAll(OUTBOX)).Methods("GET")
	router.HandleFunc("/email/outbox/{uuid}", MSADelete(OUTBOX)).Methods("DELETE")

	// Client methods
	router.HandleFunc("/email", MSASend).Methods("POST")
	router.HandleFunc("/email", MSAReadAll(INBOX)).Methods("GET")
	router.HandleFunc("/email/{uuid}", MSARead).Methods("GET")
	router.HandleFunc("/email/{uuid}", MSADelete(INBOX)).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8888", router))
}

// MSASend gets called from the handleRequests method
// It places the raw JSON in the outbox with a UUID, for the MTA to pick
// it up and handle it
func MSASend(w http.ResponseWriter, r *http.Request) {

	var email EMail
	rootpath := "Outbox"

	//Create a UUID for the message
	uuid, err := uuid.NewUUID()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	filename := uuid.String() + ".email"
	path := filepath.Join(rootpath, filename)

	//Write the JSON to a file, whose name is the UUID
	bodyBytes, err := ioutil.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}

	err = json.Unmarshal(bodyBytes, &email)

	if err != nil {
		// Assume the email is badly formatted and exit
		w.WriteHeader(http.StatusBadRequest)
		log.Print(err.Error())
		return
	} else if email.From == "" || email.To == "" {
		// Assume the email is badly formatted and exit
		w.WriteHeader(http.StatusBadRequest)
		log.Println("From or To field empty.")
		return
	}

	email.UUID = uuid

	bodyBytes, err = json.Marshal(email)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Panic(err.Error())
		return
	}

	if err := ioutil.WriteFile(path, bodyBytes, 0755); err == nil {
		w.WriteHeader(http.StatusCreated)
	} else {
		//Could not write the message to outbox
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
	}
}

// MSAReceive unpacks a message and writes it to the inbox
func MSAReceive(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}

	rootpath := "Inbox"

	// if there's an error while creating a UUID, we will use the existing one
	inboxUUID, err := uuid.NewUUID()

	var email EMail
	if err == nil {
		err = json.Unmarshal(body, &email)

		// if the email is badly formed, don't continue
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Print(err.Error())
			return
		}

		email.UUID = inboxUUID
	} else {
		err = json.Unmarshal(body, &email)

		// if the email is badly formed, don't continue
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Print(err.Error())
			return
		}

		inboxUUID = email.UUID
	}

	filePath := filepath.Join(rootpath, inboxUUID.String())

	// Update the email with the new UUID
	emailJSON, err := json.Marshal(email)

	// Problem marshalling the email into JSON... error and inform MTA
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}

	err = ioutil.WriteFile(filePath+".email", emailJSON, 0755)

	// if there's an error writing to inbox, inform MTA
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
	} else {
		w.WriteHeader(http.StatusOK)
	}

	log.Println("Received an email : " + email.Subject + " from : " + email.From)
}

// MSAReadAll gets called from the handleRequests method
// It reads all the messages in the Inbox folder of the specified user
// It then sends as a response the UUID and object of each message
func MSAReadAll(folder int) func(w http.ResponseWriter, r *http.Request) {

	// This is so that we can pass in an argument (folder)
	return func(w http.ResponseWriter, r *http.Request) {

		// Handle whether we are reading from Inbox or Outbox
		var rootpath string
		if folder == INBOX {
			rootpath = "Inbox"
		} else if folder == OUTBOX {
			rootpath = "Outbox"
		} else {
			// Not inbox or outbox ? Then nothing.
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// DEBUG:
		log.Println("Read all in " + rootpath)

		// Read a list of all files in the directory
		files, err := ioutil.ReadDir(rootpath)
		log.Printf("There's %d email in %s", len(files), rootpath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Put all the emails in a struct, to be formatted in JSON
		var folder Folder

		for _, file := range files {
			path := filepath.Join(rootpath, file.Name())
			fmt.Println(path)
			var email EMail
			if data, err := ioutil.ReadFile(path); err == nil {

				err = json.Unmarshal(data, &email)

				if err != nil {
					log.Println(err.Error())
					continue
				}

				log.Printf("%+v\n", email)
				folder.Emails = append(folder.Emails, email)
			} else {

				if os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Print(err.Error())
					return
				} else if os.IsPermission(err) {
					w.WriteHeader(http.StatusForbidden)
					log.Print(err.Error())
					return
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					log.Print(err.Error())
					return
				}
			}
		}

		// Format the fodler as JSON
		folderJSON, err := json.Marshal(folder)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err.Error())
		}

		// Send the response
		w.Write(folderJSON)
	}
}

// MSARead gets called from the handleRequests method
// It reads a specific message in the inbox
// It then returns the email metadata and contents
func MSARead(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	uuid := vars["uuid"]

	rootpath := "Inbox"

	filename := uuid + ".email"
	path := filepath.Join(rootpath, filename)
	log.Print(path)

	// Read the data contained in the email
	files, _ := filepath.Glob("Inbox/*.*")
	log.Print(files)

	if data, err := ioutil.ReadFile(path); err == nil {

		// Send email data back to user
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	} else if err != nil {
		log.Print(err.Error())

		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else if os.IsPermission(err) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// MSADelete gets called from the handleRequests method
// It deletes a specific message in the user's Inbox
func MSADelete(folder int) func(w http.ResponseWriter, r *http.Request) {

	// This is so that we can pass in an argument (folder)
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		uuid := vars["uuid"]

		// Handle whether we are reading from Inbox or Outbox
		var rootpath string
		if folder == INBOX {
			rootpath = "Inbox"
		} else if folder == OUTBOX {
			rootpath = "Outbox"
		} else {
			// Not inbox or outbox ? Then nothing.
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("Delete email %s in %s\n", uuid, rootpath)

		filename := uuid + ".email"
		path := filepath.Join(rootpath, filename)

		// Delete the email
		if err := os.Remove(path); err == nil {
			w.WriteHeader(http.StatusOK)
		} else {

			// Or tell us what happened if we can't !
			if os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
			} else if os.IsPermission(err) {
				w.WriteHeader(http.StatusForbidden)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}
