package main

import (
	"errors"
	"fmt"
)

type organization struct {
	Name string
	err  error
}

func allOrganizations(s *server) (orgs []organization, err error) {
	rows, err := s.DB.Query("SELECT name FROM organizations")
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

func (o *organization) save(s *server) (err error) {
	const nameMaxLen = 128
	const nameMinLen = 3
	if len(o.Name) > nameMaxLen {
		o.err = fmt.Errorf("Name must be less than %v characters long", nameMaxLen+1)
		return
	}
	if len(o.Name) < nameMinLen {
		o.err = fmt.Errorf("Name must be %v or more characters long", nameMinLen)
		return
	}
	if _, err = s.DB.Exec("INSERT INTO organizations (name) VALUES ($1)", o.Name); err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"organizations_pkey\"" {
			o.err = errors.New("That name has already been taken")
			return nil
		}
		return
	}
	return
}
