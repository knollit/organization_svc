package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/golang/protobuf/proto"
	_ "github.com/lib/pq"
	orgPB "github.com/mikeraimondi/api_service/organizations/proto"
)

var (
	dbAddr = flag.String("db-addr", os.Getenv("POSTGRES_PORT_5432_TCP_ADDR"), "Database address")
	dbPW   = flag.String("db-pw", os.Getenv("POSTGRES_PASSWORD"), "Database password")
)

func main() {
	connStr := fmt.Sprintf("user=postgres host=%v password=%v dbname=postgres sslmode=disable", *dbAddr, *dbPW)
	db, _ := sql.Open("postgres", connStr)
	server := &server{
		DB: db,
	}
	defer func() {
		if err := server.Close(); err != nil {
			log.Println("Failed to close server: ", err)
		}
	}()

	log.Fatal(server.run(":80"))
}

type server struct {
	DB *sql.DB
}

func (s *server) handler() http.Handler {
	return s.rootHandler()
}

func (s *server) run(addr string) error {
	if err := s.DB.Ping(); err != nil {
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

func (s *server) rootHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			orgs, err := allOrganizations(s)
			if err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			orgspb := []*orgPB.Organization{}
			for _, o := range orgs {
				orgspb = append(orgspb, &orgPB.Organization{Name: *proto.String(o.Name)})
			}
			orgsMsg := &orgPB.Organizations{
				Orgs: orgspb,
			}
			data, err := proto.Marshal(orgsMsg)
			if err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			w.Write(data)
			return
		} else if r.Method == "POST" {
			buf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			orgMsg := &orgPB.Organization{}
			err = proto.Unmarshal(buf, orgMsg)
			if err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}

			org := organization{Name: orgMsg.Name}
			if err := org.save(s); err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			if org.err != nil {
				orgMsg.Error = *proto.String(org.err.Error())
			}
			data, err := proto.Marshal(orgMsg)
			if err != nil {
				log.Print(err)
				http.Error(w, "Internal application error", http.StatusInternalServerError)
				return
			}
			w.Write(data)
			return
		} else {
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
