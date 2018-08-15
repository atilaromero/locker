package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

func (l *lock) lockPathAndRespond(p string, w http.ResponseWriter) {
	if ok := l.lockPath(p); !ok {
		w.WriteHeader(http.StatusLocked)
		fmt.Fprintf(w, "resource already locked\n")
		return
	}
	fmt.Printf("just locked: %v\n", p)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
}

func (l *lock) unlockPathAndRespond(p string, w http.ResponseWriter) {
	l.unlockPath(p)
	fmt.Printf("unlocked   : %v\n", p)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
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
	r.HandleFunc("/unlock/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		p := r.FormValue("path")
		l.unlockPathAndRespond(p, w)
	}).Methods("GET")
	r.HandleFunc("/lock/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		p := r.FormValue("path")
		l.lockPathAndRespond(p, w)
	}).Methods("GET")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "text/html")
		for k := range l.lockedPaths {
			fmt.Fprintf(w, "%s\n", k)
			fmt.Fprintf(w, "<a href='./unlock/?path=%s'>Unlock</a>\n", url.QueryEscape(k))
			fmt.Fprintf(w, "<br>\n")
		}
	}).Methods("GET")

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
			p := event.Payload.EvidencePath
			l.lockPathAndRespond(p, w)
		case "UNLOCK":
			p := event.Payload.EvidencePath
			l.unlockPathAndRespond(p, w)
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "event type not known: %v\n", event.Type)
		}
	}
}
