package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type server struct {
	DB *sql.DB
}

func (s *server) handler() http.Handler {
	return s.rootHandler()
}

func (s *server) run(addr string) error {
	dbAddr := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
	dbPW := os.Getenv("POSTGRES_PASSWORD")
	db, err := sql.Open("postgres", "user=postgres password="+dbPW+" host="+dbAddr+" dbname=postgres sslmode=disable")
	if err != nil {
		return err
	}
	s.DB = db
	if err = s.DB.Ping(); err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:    addr,
		Handler: s.handler(),
	}

	log.Printf("Listening for requests on %s...\n", addr)
	return httpServer.ListenAndServe()
}

func (s *server) Close() error {
	if err := s.DB.Close(); err != nil {
		log.Println("Failed to close database connection cleanly: ", err)
	}

	return nil
}

func main() {
	server := &server{}
	defer func() {
		if err := server.Close(); err != nil {
			log.Println("Failed to close server: ", err)
		}
	}()

	log.Fatal(server.run(":80"))
}

func (s *server) rootHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			rows, err := s.DB.Query("SELECT name FROM organizations")
			if err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var names []string
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					log.Print(err)
					http.Error(w, "Internal application error", http.StatusInternalServerError)
					return
				}
				names = append(names, name)
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(names)
			return
		} else if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Error parsing form", http.StatusBadRequest)
				return
			}
			if len(r.Form["name"]) != 1 {
				http.Error(w, "Invalid data", http.StatusBadRequest)
				return
			}
			if _, err := s.DB.Exec("INSERT INTO organizations (name) VALUES ($1)", r.Form["name"][0]); err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		} else {
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
