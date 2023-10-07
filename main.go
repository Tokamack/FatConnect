package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

const (
	dbHost     = "localhost"
	dbPort     = "5432"
	dbUser     = "postgres"
	dbPassword = "toka"
	dbName     = "postgres"
)

func handleRegistration(w http.ResponseWriter, r *http.Request) {
	// Подключение к базе данных PostgreSQL.
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Разбор JSON из запроса.
	var user User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Выполнение SQL-запроса для вставки новой записи в таблицу "users".
	_, err = db.Exec("INSERT INTO users (name) VALUES ($1)", user.Name)
	if err != nil {
		http.Error(w, "Failed to insert user", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Отправка успешного ответа.
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "User %s with ID %d registered successfully\n", user.Name)
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	// Подключение к базе данных PostgreSQL.
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Разбор JSON из запроса.
	var inputID struct {
		ID int `json:"id"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&inputID); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Выполнение SQL-запроса для получения "name" по "id" из таблицы "users".
	var name string
	err = db.QueryRow("SELECT name FROM users WHERE id = $1", inputID.ID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve user name", http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}

	// Отправка "name" в ответе.
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/reg/new", handleRegistration)
	http.HandleFunc("/reg/auth", handleAuth)
	fmt.Println("START")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
