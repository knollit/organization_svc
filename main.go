package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"

	"github.com/google/flatbuffers/go"
	_ "github.com/lib/pq"

	"github.com/mikeraimondi/knollit/organization_svc/organizations"
	"github.com/mikeraimondi/prefixedio"
)

var (
	dbAddr   = flag.String("db-addr", os.Getenv("POSTGRES_PORT_5432_TCP_ADDR"), "Database address")
	dbPW     = flag.String("db-pw", os.Getenv("POSTGRES_PASSWORD"), "Database password")
	caPath   = flag.String("ca-path", os.Getenv("TLS_CA_PATH"), "Path to CA file")
	certPath = flag.String("cert-path", os.Getenv("TLS_CERT_PATH"), "Path to cert file")
	keyPath  = flag.String("key-path", os.Getenv("TLS_KEY_PATH"), "Path to private key file")
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	connStr := fmt.Sprintf("user=postgres host=%v password=%v dbname=postgres sslmode=disable", *dbAddr, *dbPW)
	db, _ := sql.Open("postgres", connStr)

	// Load server cert
	cert, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		logger.Fatal("Failed to open server cert and/or key: ", err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*caPath)
	if err != nil {
		logger.Fatal("Failed to open CA cert: ", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		logger.Fatal("Failed to parse CA cert")
	}

	server := newServer()
	server.logger = logger
	server.db = db
	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          caCertPool,
		InsecureSkipVerify: true, //TODO dev only
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	server.listenFunc = func(addr string) (net.Listener, error) {
		return tls.Listen("tcp", addr, tlsConf)
	}
	defer func() {
		if err := server.Close(); err != nil {
			server.logger.Println("Failed to close server: ", err)
		}
	}()

	server.logger.Fatal(server.run(":13800"))
}

func newServer() *server {
	return &server{
		builderPool: sync.Pool{
			New: func() interface{} {
				return flatbuffers.NewBuilder(0)
			},
		},
		prefixedBufPool: sync.Pool{
			New: func() interface{} {
				return &prefixedio.Buffer{}
			},
		},
	}
}

type DB interface {
	Begin() (*sql.Tx, error)
	Close() error
	Driver() driver.Driver
	Exec(query string, args ...interface{}) (sql.Result, error)
	Ping() error
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	SetMaxIdleConns(n int)
	SetMaxOpenConns(n int)
	Stats() sql.DBStats
}

type server struct {
	db              DB
	listenFunc      func(string) (net.Listener, error)
	logger          *log.Logger
	ready           chan int
	builderPool     sync.Pool
	prefixedBufPool sync.Pool
}

func (s *server) handler(conn net.Conn) {
	defer conn.Close()

	buf := s.prefixedBufPool.Get().(*prefixedio.Buffer)
	defer s.prefixedBufPool.Put(buf)
	_, err := buf.ReadFrom(conn)
	if err != nil {
		s.logger.Print(err)
		// TODO send error
		return
	}
	req := organizations.GetRootAsOrganization(buf.Bytes(), 0)

	b := s.builderPool.Get().(*flatbuffers.Builder)
	defer s.builderPool.Put(b)
	switch req.Action() {
	case organizations.ActionIndex:
		orgs, err := allOrganizations(s.db)
		if err != nil {
			s.logger.Print(err)
			// TODO send error
			return
		}
		for _, o := range orgs {
			if _, err := prefixedio.WriteBytes(conn, o.toFlatBufferBytes(b)); err != nil {
				s.logger.Print(err)
			}
		}
	case organizations.ActionNew:
		org := organizationFromFlatBuffer(req)
		if err := org.save(s); err != nil {
			s.logger.Print(err)
			// TODO send error
			return
		}
		if _, err := prefixedio.WriteBytes(conn, org.toFlatBufferBytes(b)); err != nil {
			s.logger.Print(err)
		}
	case organizations.ActionRead:
		org, err := organizationByName(s.db, string(req.Name()))
		if err != nil {
			// Do something
			return
		}
		if _, err := prefixedio.WriteBytes(conn, org.toFlatBufferBytes(b)); err != nil {
			s.logger.Print(err)
		}
	}
	return
}

func (s *server) run(addr string) error {
	if err := s.db.Ping(); err != nil {
		return err
	}
	listener, err := s.listenFunc(addr)
	if err != nil {
		return err
	}
	s.listenFunc = func(s string) (net.Listener, error) {
		return listener, nil
	}

	s.logger.Printf("Listening for requests on %s...\n", addr)
	if s.ready != nil {
		s.ready <- 1
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handler(conn)
	}
}

func (s *server) Close() error {
	listener, _ := s.listenFunc("")
	if err := listener.Close(); err != nil {
		s.logger.Println("Failed to close TCP listener cleanly: ", err)
	}
	if err := s.db.Close(); err != nil {
		s.logger.Println("Failed to close database connection cleanly: ", err)
	}

	return nil
}
