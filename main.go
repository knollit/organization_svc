package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/golang/protobuf/proto"
	_ "github.com/lib/pq"

	"github.com/mikeraimondi/api_service"
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

	log.Fatal(server.run(":13800"))
}

type server struct {
	DB       *sql.DB
	listener net.Listener
}

func (s *server) handler(conn net.Conn) {
	defer conn.Close()
	buf, err := apiService.ReadWithSize(conn)
	if err != nil {
		log.Print(err)
		// TODO send error
		return
	}
	req := &orgPB.Request{}
	if err := proto.Unmarshal(buf, req); err != nil {
		log.Print(err)
		// TODO send error
		return
	}

	if req.Action == orgPB.Request_INDEX {
		orgs, err := allOrganizations(s)
		if err != nil {
			log.Print(err)
			// TODO send error
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
			// TODO send error
			return
		}
		apiService.WriteWithSize(conn, data)
		return
	} else if req.Action == orgPB.Request_NEW {
		org := organization{Name: req.Organization.Name}
		if err := org.save(s); err != nil {
			log.Print(err)
			// TODO send error
			return
		}
		orgMsg := &orgPB.Organization{}
		if org.err != nil {
			orgMsg.Error = *proto.String(org.err.Error())
		}
		data, err := proto.Marshal(orgMsg)
		if err != nil {
			log.Print(err)
			// TODO send error
			return
		}
		apiService.WriteWithSize(conn, data)
		return
	}
}

func (s *server) run(addr string) error {
	if err := s.DB.Ping(); err != nil {
		return err
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = l

	log.Printf("Listening for requests on %s...\n", addr)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handler(conn)
	}
}

func (s *server) Close() error {
	if err := s.listener.Close(); err != nil {
		log.Println("Failed to close TCP listener cleanly: ", err)
	}
	if err := s.DB.Close(); err != nil {
		log.Println("Failed to close database connection cleanly: ", err)
	}

	return nil
}
