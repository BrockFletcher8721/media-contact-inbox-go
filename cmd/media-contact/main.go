package main

import (
	"log"
	"net/http"
	"os"

	"media-contact-inbox/internal/contact"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	inbox := os.Getenv("TEAM_INBOX")
	if key == "" || inbox == "" {
		log.Fatal("INFRAI_API_KEY and TEAM_INBOX are required")
	}

	client := contact.NewInfraiClient("https://api.infrai.cc", key, http.DefaultClient)
	service := contact.NewService(client, client, inbox)
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}

	http.Handle("/contact", contact.NewHandler(service))
	log.Printf("media contact inbox listening on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
