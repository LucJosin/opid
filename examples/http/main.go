// Command http serves local endpoints to generate, encode, and decode OPIDs.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"uuid"

	"github.com/LucJosin/opid"
)

const (
	resourcePrefix = "usr"
	address        = "127.0.0.1:8080"
	serverTimeout  = 5 * time.Second
)

type response struct {
	ID    opid.ID `json:"id,omitempty"`
	UUID  string  `json:"uuid,omitempty"`
	Error string  `json:"error,omitempty"`
}

type api struct {
	codec *opid.Codec
}

func main() {
	codec, err := opid.New(resourcePrefix)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:         address,
		Handler:      newHandler(codec),
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
		IdleTimeout:  time.Minute,
	}

	log.Printf("Listening on http://%s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newHandler(codec *opid.Codec) http.Handler {
	app := &api{codec: codec}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+resourcePrefix+"/generate", app.generate)
	mux.HandleFunc("GET /"+resourcePrefix+"/encode/{uuid}", app.encode)
	mux.HandleFunc("GET /"+resourcePrefix+"/decode/{id}", app.decode)
	return mux
}

func (app *api) generate(w http.ResponseWriter, r *http.Request) {
	id := uuid.NewV7()
	publicID, err := app.codec.Encode(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{Error: fmt.Sprintf("Could not generate ID: %v", err)})
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, response{ID: publicID, UUID: id.String()})
}

func (app *api) encode(w http.ResponseWriter, r *http.Request) {
	publicID, err := app.codec.EncodeString(r.PathValue("uuid"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "Expected a UUIDv7"})
		return
	}

	writeJSON(w, http.StatusOK, response{ID: publicID})
}

func (app *api) decode(w http.ResponseWriter, r *http.Request) {
	id, err := app.codec.DecodeString(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: fmt.Sprintf("Invalid public ID: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, response{UUID: id.String()})
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("Write response: %v", err)
	}
}
