package main

import (
	"flag"
	"net"
	"os"
	"testing"

	"github.com/google/flatbuffers/go"
	"github.com/knollit/coelacanth"
	ct "github.com/knollit/coelacanth/testing"
	"github.com/knollit/organization_svc/organizations"
	"github.com/mikeraimondi/prefixedio"
)

func TestMain(m *testing.M) {
	flag.Parse()
	exitCode := m.Run()
	ct.RunAfterCallbacks()
	os.Exit(exitCode)
}

func TestOrganizationIndexWithOne(t *testing.T) {
	ct.RunWithServer(t, handler, func(s *coelacanth.Server, addr string) {
		// Test-specific setup
		const name = "testOrg"
		if _, err := s.DB.Exec("INSERT INTO organizations (name) VALUES ($1)", name); err != nil {
			t.Fatal(err)
		}

		// Begin test
		conn, err := net.Dial("tcp", addr)
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

func TestOrganizationReadWithOne(t *testing.T) {
	ct.RunWithServer(t, handler, func(s *coelacanth.Server, addr string) {
		org := organization{
			ID:   "5ff0fcbd-8b51-11e5-a171-df11d9bd7d62",
			Name: "testOrg",
		}
		if _, err := s.DB.Exec("INSERT INTO organizations (id, name) VALUES ($1, $2)", org.ID, org.Name); err != nil {
			t.Fatal("error saving org: ", err)
		}

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := flatbuffers.NewBuilder(0)
		orgReq := organization{
			Name:   org.Name,
			action: organizations.ActionRead,
		}
		if _, err := prefixedio.WriteBytes(conn, orgReq.toFlatBufferBytes(b)); err != nil {
			t.Fatal(err)
		}

		var buf prefixedio.Buffer
		_, err = buf.ReadFrom(conn)
		if err != nil {
			t.Fatal("error reading response from server: ", err)
		}
		orgRes := organizations.GetRootAsOrganization(buf.Bytes(), 0)

		if resName := string(orgRes.Name()); resName != org.Name {
			t.Fatalf("received organization name doesn't match test name. expected: %q. actual: %q\n", org.Name, resName)
		}
		if resID := string(orgRes.ID()); resID != org.ID {
			t.Fatalf("received organization ID doesn't match test ID. expected: %q. actual: %q\n", org.ID, resID)
		}
	})
}
