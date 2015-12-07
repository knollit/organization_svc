package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/knollit/http_frontend/organizations"
	"github.com/mikeraimondi/prefixedio"
)

var (
	afterCallbacks []func() error
	commonDB       *sql.DB
)

func TestMain(m *testing.M) {
	flag.Parse()
	exitCode := m.Run()
	for _, cb := range afterCallbacks {
		if err := cb(); err != nil {
			fmt.Println(err)
		}
	}
	os.Exit(exitCode)
}

func registerAfterCallback(id string, cb func() error) {
	afterCallbacks = append(afterCallbacks, cb)
}

type logWriter struct {
	*testing.T
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.Log(string(p))
	return len(p), nil
}

type testDB struct {
	DB
}

func (db testDB) Close() error {
	return nil
}

func runWithDB(t *testing.T, testFunc func(testDB)) {
	if commonDB == nil {
		db, _ := sql.Open("postgres", "user=mike host=localhost dbname=postgres sslmode=disable")
		if err := db.Ping(); err != nil {
			t.Fatal("Error opening DB: ", err)
		}
		db.Exec("DROP DATABASE IF EXISTS endpoints_test")
		db.Exec("CREATE DATABASE endpoints_test")
		db.Close()
		commonDB, _ = sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")
		registerAfterCallback("closeDB", func() error {
			return commonDB.Close()
		})
	}

	testDB := testDB{
		DB: commonDB,
	}
	setupSQL, _ := ioutil.ReadFile("db/db.sql")
	if _, err := testDB.Exec(string(setupSQL)); err != nil {
		t.Fatal("Error setting up DB: ", err)
	}
	if _, err := testDB.Exec("BEGIN"); err != nil {
		t.Fatal("Error starting TX: ", err)
	}
	defer func() {
		if _, err := testDB.Exec("ROLLBACK"); err != nil {
			t.Fatal("Error rolling back TX: ", err)
		}
	}()
	testFunc(testDB)
	return
}

func runWithServer(t *testing.T, testFunc func(*server)) {
	runWithDB(t, func(db testDB) {
		// Setup server
		rdy := make(chan int)
		s := newServer()
		defer func() {
			if err := s.Close(); err != nil {
				t.Fatal("Error closing server: ", err)
			}
		}()

		s.db = db
		s.listenFunc = func(addr string) (net.Listener, error) {
			return net.Listen("tcp", addr)
		}
		s.ready = rdy
		s.logger = log.New(&logWriter{t}, "", log.Lmicroseconds)

		errs := make(chan error)
		go func() {
			errs <- s.run(":13900") // TODO not hardcoded
		}()
		select {
		case err := <-errs:
			t.Fatal(err)
		case <-time.NewTimer(10 * time.Second).C:
			t.Fatal("Timed out waiting for server to start")
		case <-rdy:
			testFunc(s)
		}
	})
	return
}

func TestEndpointIndexWithOne(t *testing.T) {
	runWithServer(t, func(s *server) {
		// Test-specific setup
		const name = "testOrg"
		if _, err := s.db.Exec("INSERT INTO organizations (name) VALUES ($1)", name); err != nil {
			t.Fatal(err)
		}

		// Begin test
		listener, _ := s.listenFunc("")
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := flatbuffers.NewBuilder(0)
		orgReq := organization{
			Name:   name,
			action: organizations.ActionRead,
		}
		if _, err := prefixedio.WriteBytes(conn, orgReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		var buf prefixedio.Buffer
		_, err = buf.ReadFrom(conn)
		if err != nil {
			t.Fatal("Error reading response from server: ", err)
		}
		orgRes := organizations.GetRootAsOrganization(buf.Bytes(), 0)

		if resName := string(orgRes.Name()); resName != name {
			t.Fatalf("Received organization name doesn't match test name. Expected: %q. Actual: %q\n", name, resName)
		}
	})
}
