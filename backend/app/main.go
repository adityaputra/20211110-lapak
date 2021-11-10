package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	//"context"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/go-redis/redis"

)

func connect() (*sql.DB, error) {
	addr := os.Getenv("MYSQL_ADDRESS")
	user := os.Getenv("MYSQL_USER")
	pass := os.Getenv("MYSQL_PASS")
	db := os.Getenv("MYSQL_DB")
	return sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", string(user), string(pass), string(addr), string(db)))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	enabled := os.Getenv("MYSQL_ENABLED")
	if enabled != "true" {
		fmt.Fprintf(w, "ERROR: MySQL is disabled")
		return
	}
	db, err := connect()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	switch r.Method {
		case "POST":
			decoder := json.NewDecoder(r.Body)
			type LapakInput struct {
				Name string
				Owner string
				Products_Sold int
			}
			var t LapakInput 
			err := decoder.Decode(&t)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(w, t.Name)

                if _, err := db.Exec("INSERT INTO lapak (lapak_name, lapak_owner, products_sold) VALUES (?, ?, ?);", (t.Name), (t.Owner), (t.Products_Sold)); err != nil {
                        return
                }

		case "GET":
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
		default:
			fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func cacheHandler(w http.ResponseWriter, r *http.Request) {
	enabled := os.Getenv("REDIS_ENABLED")
	if enabled != "true" {
		fmt.Fprintf(w, "ERROR: Redis is disabled")
		return
	}
	type Cache struct {
		Key string
		Value string
	}
	switch r.Method {
		case "POST":
			var c Cache
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&c)
			if err != nil {
				panic(err)
			}
			fmt.Println(c)
			if c.Key == "" {
				fmt.Fprintf(w, "ERROR: key variable is required")
				return
			}
			if c.Value == "" {
				fmt.Fprintf(w, "ERROR: value variable is required")
				return
			}

			addr := os.Getenv("REDIS_ADDRESS") 
			fmt.Println("Go Redis Connection Test")
			client := redis.NewClient(&redis.Options{
				Addr: fmt.Sprintf("%s:6379", addr),
				Password: "",
				DB: 0,
			})

			insert := client.Set(c.Key, c.Value, 0)
			fmt.Println(insert)
			fmt.Fprintf(w, "ok")
		case "GET":
			var c Cache
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&c)
			if err != nil {
				panic(err)
			}
			fmt.Println(c)
			if c.Key == "" {
				fmt.Fprintf(w, "ERROR: key variable is required")
				return
			}

			addr := os.Getenv("REDIS_ADDRESS") 
			fmt.Println("Go Redis Connection Test")
			client := redis.NewClient(&redis.Options{
				Addr: fmt.Sprintf("%s:6379", addr),
				Password: "",
				DB: 0,
			})

			get := client.Get(c.Key)
			fmt.Println(get.Val())
			json.NewEncoder(w).Encode(get.Val())
		default:
			fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func cacheListHandler(w http.ResponseWriter, r *http.Request) {
	enabled := os.Getenv("REDIS_ENABLED")
	if enabled != "true" {
		fmt.Fprintf(w, "ERROR: Redis is disabled")
		return
	}
	switch r.Method {
		case "GET":
			addr := os.Getenv("REDIS_ADDRESS") 
			fmt.Println("Go Redis Connection Test")
			client := redis.NewClient(&redis.Options{
				Addr: fmt.Sprintf("%s:6379", addr),
				Password: "",
				DB: 0,
			})

			pong, err := client.Ping().Result()
			fmt.Println(pong, err)

			//ctx := context.Background()
			//val, err := client.Get("key").Result()
			//fmt.Println(val)
			var keys []string
			iter := client.Scan(0, "", 0).Iterator()
			for iter.Next() {
				fmt.Println("iterating")
				fmt.Println("keys", iter.Val())
				keys = append(keys, iter.Val())
			}
			if err := iter.Err(); err != nil {
				panic(err)
			}
			json.NewEncoder(w).Encode(keys)

		default:
			fmt.Fprintf(w, "Sorry, only GET method is supported.")
	}
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
	r.HandleFunc("/cache", cacheHandler)
	r.HandleFunc("/cache/list", cacheListHandler)
	port := os.Getenv("LISTEN_PORT") 
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handlers.LoggingHandler(os.Stdout, r)))
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
