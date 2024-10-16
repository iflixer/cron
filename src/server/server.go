package server

import (
	"encoding/json"
	"fmt"
	"local/database/flixcron"
	"log"
	"net/http"
)

type Service struct {
	port        string
	server      http.Server
	cronService *flixcron.Service
}

func (s *Service) Run() {
	addr := fmt.Sprintf(":%s", s.port)
	log.Println("Starting server on", addr)
	err := s.server.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func NewService(port string, cronService *flixcron.Service) (s *Service, err error) {

	s = &Service{
		port:        port,
		cronService: cronService,
	}
	s.server = http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: http.HandlerFunc(s.Handler),
	}
	return
}

func (s *Service) Handler(w http.ResponseWriter, r *http.Request) {

	if r.URL.String() == "/health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	if r.URL.String() == "/export" {
		crons := s.cronService.Export()
		j, _ := json.Marshal(crons)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-type", "application/json")
		w.Write(j)
		return
	}

}
