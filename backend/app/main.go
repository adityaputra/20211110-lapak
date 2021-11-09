package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func connect() (*sql.DB, error) {
	bin, err := ioutil.ReadFile("/run/secrets/db-password")
	if err != nil {
		return nil, err
	}
	return sql.Open("mysql", fmt.Sprintf("root:%s@tcp(db:3306)/example", string(bin)))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connect()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM lapak")
	if err != nil {
		w.WriteHeader(500)
		return
	}
	type Lapak struct {
		Id int
		LapakName string
		LapakOwner string
		ProductsSold int
	}
	var lapaks []*Lapak
	for rows.Next() {
		lapak := new(Lapak)
		err := rows.Scan(&lapak.Id, &lapak.LapakName, &lapak.LapakOwner, &lapak.ProductsSold)
		if err != nil {
			fmt.Println(err)
			return
		}
		lapaks = append(lapaks, lapak)
	}
	json.NewEncoder(w).Encode(lapaks)
}

func main() {
	log.Print("Prepare db...")
	if err := prepare(); err != nil {
		log.Fatal(err)
	}

	log.Print("Listening 8000")
	r := mux.NewRouter()
	r.HandleFunc("/", mainHandler)
	r.HandleFunc("/db", mainHandler)
	log.Fatal(http.ListenAndServe(":8000", handlers.LoggingHandler(os.Stdout, r)))
}

func prepare() error {
	db, err := connect()
	if err != nil {
		return err
	}
	defer db.Close()

	for i := 0; i < 60; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if _, err := db.Exec("DROP TABLE IF EXISTS lapak"); err != nil {
		return err
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS lapak (id INT NOT NULL AUTO_INCREMENT, PRIMARY KEY(id), lapak_name VARCHAR(256) NOT NULL, lapak_owner VARCHAR(256) NOT NULL, products_sold INT NOT NULL);"); err != nil {
		return err
	}
        for i := 0; i < 5; i++ {
                if _, err := db.Exec("INSERT INTO lapak (lapak_name, lapak_owner, products_sold) VALUES (?, ?, ?);", fmt.Sprintf("lapak%d", i), fmt.Sprintf("budi%d", i), fmt.Sprintf("%d", i)); err != nil {
                        return err
                }
        }

	return nil
}
