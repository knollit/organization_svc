package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	addr := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
	pw := os.Getenv("POSTGRES_PASSWORD")
	var err error
	db, err = sql.Open("postgres", "user=postgres password="+pw+" host="+addr+" dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal("Failure opening database: ", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failure connecting to database: ", err)
	}
}

func main() {
	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT name FROM organizations")
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
		if _, err := db.Exec("INSERT INTO organizations (name) VALUES ($1)", r.Form["name"][0]); err != nil {
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
}
