package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

type lock struct {
	sync.Mutex
	lockedPaths map[string]bool
}

func (l *lock) lockPath(p string) bool {
	l.Lock()
	defer l.Unlock()
	if _, ok := l.lockedPaths[p]; ok {
		return false
	}
	l.lockedPaths[p] = true
	return true
}

func (l *lock) unlockPath(p string) {
	l.Lock()
	defer l.Unlock()
	delete(l.lockedPaths, p)
}

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "80"
	}

	l := lock{
		lockedPaths: make(map[string]bool),
	}

	r := mux.NewRouter()
	r.HandleFunc("/", handler(&l)).Methods("POST")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("could not start server: %v\n", err)
	}
}
func handler(l *lock) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		event := struct {
			Type    string `json:"type"`
			Payload struct {
				EvidencePath string `json:"evidencePath"`
			} `json:"payload"`
		}{}
		err := decoder.Decode(&event)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "error decoding request: %v\n", err)
			return
		}

		switch event.Type {
		case "LOCK":
			if ok := l.lockPath(event.Payload.EvidencePath); !ok {
				w.WriteHeader(http.StatusLocked)
				fmt.Fprintf(w, "resource already locked\n")
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "ok")
		case "UNLOCK":
			l.unlockPath(event.Payload.EvidencePath)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "ok")
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "event type not known: %v\n", event.Type)
		}
	}
}
