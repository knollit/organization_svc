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
	"github.com/mikeraimondi/coelacanth"
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

func registerAfterCallback(cb func() error) {
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
	coelacanth.DB
	testTx *sql.Tx
}

func (db testDB) Begin() (*sql.Tx, error) {
	return db.testTx, nil
}

func (db testDB) Close() error {
	return nil
}

func (db testDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.testTx.Exec(query, args...)
}

func (db testDB) Prepare(query string) (*sql.Stmt, error) {
	return db.testTx.Prepare(query)
}

func (db testDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.testTx.Query(query, args...)
}

func (db testDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.testTx.QueryRow(query, args...)
}

func runWithDB(t *testing.T, testFunc func(*testDB)) {
	if commonDB == nil {
		db, _ := sql.Open("postgres", "user=mike host=localhost dbname=postgres sslmode=disable")
		if err := db.Ping(); err != nil {
			t.Fatal("Error opening DB: ", err)
		}
		db.Exec("DROP DATABASE IF EXISTS endpoints_test")
		db.Exec("CREATE DATABASE endpoints_test")
		db.Close()
		commonDB, _ = sql.Open("postgres", "user=mike host=localhost dbname=endpoints_test sslmode=disable")
		registerAfterCallback(func() error {
			return commonDB.Close()
		})
	}

	testDB := &testDB{
		DB: commonDB,
	}
	setupSQL, _ := ioutil.ReadFile("db/db.sql")
	if _, err := testDB.DB.Exec(string(setupSQL)); err != nil {
		t.Fatal("Error setting up DB: ", err)
	}
	tx, err := testDB.DB.Begin()
	if err != nil {
		t.Fatal("Error starting TX: ", err)
	}
	testDB.testTx = tx
	defer func() {
		if err := tx.Rollback(); err != nil {
			t.Fatal("Error rolling back TX: ", err)
		}
	}()
	testFunc(testDB)
	return
}

func runWithServer(t *testing.T, testFunc func(*coelacanth.Server)) {
	runWithDB(t, func(db *testDB) {
		// Setup server
		rdy := make(chan int)
		conf := &coelacanth.Config{
			DB: db,
			ListenerFunc: func(addr string) (net.Listener, error) {
				l, err := net.Listen("tcp", addr)
				if err == nil {
					rdy <- 1
				}
				return l, err
			},
			Logger: log.New(&logWriter{t}, "", log.Lmicroseconds),
		}
		s := coelacanth.NewServer(conf)
		defer func() {
			if err := s.Close(); err != nil {
				t.Fatal("Error closing server: ", err)
			}
		}()

		// Run server on a separate goroutine
		errs := make(chan error)
		go func() {
			errs <- s.Run(":13900", handler) // TODO not hardcoded
		}()
		select {
		case err := <-errs:
			t.Fatal(err)
		case <-time.NewTimer(5 * time.Second).C:
			t.Fatal("Timed out waiting for server to start")
		case <-rdy:
			testFunc(s)
		}
	})
	return
}

func TestEndpointIndexWithOne(t *testing.T) {
	runWithServer(t, func(s *coelacanth.Server) {
		// Test-specific setup
		const name = "testOrg"
		if _, err := s.DB.Exec("INSERT INTO organizations (name) VALUES ($1)", name); err != nil {
			t.Fatal(err)
		}

		// Begin test
		conn, err := net.Dial("tcp", s.GetAddr())
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
