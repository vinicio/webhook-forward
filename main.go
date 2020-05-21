package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/vinicio/webhook-forward/forward"
)

// b:master,l:team/backend=webhook-do-backend.com&b:frontend,l:team/backend=webhook-do-frontend.com

func main() {
	r := chi.NewRouter()

	r.Post("/", forward.WebhookForward)

	log.Print("Listening on :9090")
	if err := http.ListenAndServe(":9090", r); err != nil {
		log.Fatalf("could not listen: %v", err)
	}
}
