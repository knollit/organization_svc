package main

import (
	"flag"
	"net"
	"os"
	"testing"

	"github.com/google/flatbuffers/go"
	"github.com/mikeraimondi/coelacanth"
	ct "github.com/mikeraimondi/coelacanth/testing"
	"github.com/knollit/http_frontend/organizations"
	"github.com/mikeraimondi/prefixedio"
)

func TestMain(m *testing.M) {
	flag.Parse()
	exitCode := m.Run()
	ct.RunAfterCallbacks()
	os.Exit(exitCode)
}

func TestEndpointIndexWithOne(t *testing.T) {
	ct.RunWithServer(t, handler, func(s *coelacanth.Server) {
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
