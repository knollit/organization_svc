package main

import (
	"fmt"

	"github.com/google/flatbuffers/go"
	"github.com/knollit/coelacanth"
	"github.com/knollit/organization_svc/organizations"
)

type organization struct {
	Name   string
	action int8
	err    string
}

func allOrganizations(db coelacanth.DB) (orgs []organization, err error) {
	rows, err := db.Query("SELECT name FROM organizations")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return
		}
		orgs = append(orgs, organization{Name: name})
	}
	return
}

func organizationByName(db coelacanth.DB, name string) (org *organization, err error) {
	row := db.QueryRow("SELECT name FROM organizations WHERE name = $1 LIMIT 1", name)
	var dbName string
	if err = row.Scan(&dbName); err != nil {
		return
	}
	org = &organization{
		Name: dbName,
	}
	return
}

func (org *organization) save(db coelacanth.DB) (err error) {
	const nameMaxLen = 128
	const nameMinLen = 3
	if len(org.Name) > nameMaxLen {
		org.err = fmt.Sprintf("Name must be less than %v characters long", nameMaxLen+1)
		return
	}
	if len(org.Name) < nameMinLen {
		org.err = fmt.Sprintf("Name must be %v or more characters long", nameMinLen)
		return
	}
	if _, err = db.Exec("INSERT INTO organizations (name) VALUES ($1)", org.Name); err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"organizations_pkey\"" {
			org.err = "That name has already been taken"
			return nil
		}
		return
	}
	return
}

func organizationFromFlatBuffer(org *organizations.Organization) organization {
	return organization{
		Name:   string(org.Name()),
		action: org.Action(),
		err:    string(org.Error()),
	}
}

func (org *organization) toFlatBufferBytes(b *flatbuffers.Builder) []byte {
	b.Reset()

	nameOffset := b.CreateString(org.Name)

	organizations.OrganizationStart(b)

	organizations.OrganizationAddName(b, nameOffset)
	organizations.OrganizationAddAction(b, org.action)

	orgPosition := organizations.OrganizationEnd(b)
	b.Finish(orgPosition)

	return b.FinishedBytes()
}
