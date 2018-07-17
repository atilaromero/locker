package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "80"
	}

	l := lock{
		lockedPaths: make(map[string]bool),
	}

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Paths:\n")
		fmt.Fprintf(w, "	/lock?evidencePath=...\n")
		fmt.Fprintf(w, "	/unlock?evidencePath=...\n")
	}).Methods("GET")
	r.HandleFunc("/lock", getLock(&l)).Methods("GET")
	r.HandleFunc("/unlock", getUnlock(&l)).Methods("GET")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("could not start server: %v\n", err)
	}
}

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

func getLock(l *lock) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		evidencePaths, ok := r.URL.Query()["evidencePath"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "parameter not set: evidencePath")
			return
		}
		if len(evidencePaths) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "use of multiple evidence paths is not supported")
			return
		}
		evidencePath := evidencePaths[0]
		if ok := l.lockPath(evidencePath); !ok {
			w.WriteHeader(http.StatusLocked)
			fmt.Fprintf(w, "resource already locked")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ok")
	}
}

func getUnlock(l *lock) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		evidencePaths, ok := r.URL.Query()["evidencePath"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "parameter not set: evidencePath")
			return
		}
		if len(evidencePaths) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "use of multiple evidence paths is unsuported")
			return
		}
		evidencePath := evidencePaths[0]
		l.unlockPath(evidencePath)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ok")
	}
}
