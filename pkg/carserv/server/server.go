package server

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// NewHTTPServer returns a new HTTP server
func NewHTTPServer(addr string) *http.Server {
	server := newHTTPServer()
	r := mux.NewRouter()
	r.HandleFunc("/cars", server.GetCars).Methods(http.MethodGet)
	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

type httpServer struct {
	log *log.Logger
}

func newHTTPServer() *httpServer {
	return &httpServer{
		log: log.New(os.Stdout, "logs: ", log.LstdFlags),
	}
}
