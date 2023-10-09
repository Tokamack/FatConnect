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

func main() {
	http.HandleFunc("/user/add", handleUserAdd)
	http.HandleFunc("/clubs/get_list", handleClubsGetList)
	http.HandleFunc("/club/favourite_status", handleClubFavouriteStatus)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getIdByToken(token json.RawMessage, db *sql.DB, w http.ResponseWriter) int {
	var id int
	err := db.QueryRow("SELECT id FROM users WHERE token = $1", token).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve user name", http.StatusInternalServerError)
			log.Println(err)
		}
	}
	return id
}

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

func handleUserAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Подключение к базе данных PostgreSQL.
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Разбор JSON из запроса.
	var request struct {
		NickName string          `json:"nick_name"`
		Token    json.RawMessage `json:"token"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query, err := db.Query("INSERT INTO users (nick_name, token) VALUES ($1, $2);", request.NickName, request.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No items found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve", http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}
	defer query.Close()

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		status bool `json:"status"`
	}{
		status: true,
	}

	json.NewEncoder(w).Encode(response)
}

func handleClubsGetList(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Подключение к базе данных
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}

	// Разбор JSON из запроса.
	var request struct {
		Token     json.RawMessage `json:"token"`
		Page      int             `json:"page"`
		Amenities []int           `json:"amenities"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	//Get user id by token
	var id = getIdByToken(request.Token, db, w)
	defer db.Close()

	//SQL запрос
	var t string = `SELECT
    c.id,
    c.name,
    ARRAY_AGG(ci.image_url) AS image_urls,
    jsonb_build_object('latitude', l.latitude, 'longitude', l.longitude, 'address', l.address, 'city', l.city, 'metro', l.metro) AS location,
    c.rating AS score,
    c.reviewsCount,
    c.cost,
    CASE
        WHEN f.user_id IS NOT NULL THEN true
        ELSE false
    END AS isFavorite
FROM
    clubs AS c
LEFT JOIN
    club_images AS ci ON c.id = ci.club_id
INNER JOIN
    club_location AS cl ON c.id = cl.club_id
INNER JOIN
    locations AS l ON cl.location_id = l.id
LEFT JOIN
    favorites AS f ON f.user_id = $1 AND f.object_id = c.id AND f.object_type = 'club'
LEFT JOIN
    club_amenities AS cam ON c.id = cam.club_id
$2
GROUP BY
    c.id
ORDER BY
    c.id
LIMIT
    20 OFFSET $3;`

	//make amenities part of query
	var amenities string
	if len(request.Amenities) > 0 {
		amenities = "LEFT JOIN club_amenities AS cam ON c.id = cam.club_id"
		for _, amenity := range request.Amenities {
			amenities += " AND ca.amenity_id = " + string(rune(amenity))
		}
	}

	//Actual query
	query, err := db.Query(t, id, amenities, request.Page)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No items found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve", http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}
	defer query.Close()

	//Ответ

	type ClubInfo struct {
		ID        int      `json:"id"`
		Name      string   `json:"name"`
		ImageURLs []string `json:"image_urls"`
		Location  struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Address   string  `json:"address"`
			City      string  `json:"city"`
			Metro     string  `json:"metro"`
		} `json:"location"`
		Score        float64 `json:"score"`
		ReviewsCount int     `json:"reviewsCount"`
		Cost         int     `json:"cost"`
		IsFavorite   bool    `json:"isFavorite"`
	}

	w.Header().Set("Content-Type", "application/json")
	var response []ClubInfo
	for query.Next() {
		var summary ClubInfo
		var imageUrlsJSON []byte

		err := query.Scan(
			&summary.ID,
			&summary.Name,
			&imageUrlsJSON,
			&summary.Location,
			&summary.Score,
			&summary.ReviewsCount,
			&summary.Cost,
			&summary.IsFavorite,
		)

		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal(imageUrlsJSON, &summary.ImageURLs)
		if err != nil {
			log.Fatal(err)
		}

		response = append(response, summary)
	}
	json.NewEncoder(w).Encode(response)
}

func handleClubFavouriteStatus(w http.ResponseWriter, r *http.Request) {
	const (
		Club  int = 0
		Coach     = 1
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Подключение к базе данных
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}

	// Разбор JSON из запроса.
	var request struct {
		ClubId int             `json:"club_id"`
		Token  json.RawMessage `json:"token"`
		status bool            `json:"bool"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	//Get user id by token
	var id = getIdByToken(request.Token, db, w)
	defer db.Close()

	//SQL запрос
	var t string
	if request.status {
		t = "INSERT INTO favorites (user_id, object_id, object_type)\nVALUES ($1, $2, $3);"
	} else {
		t = "DELETE FROM favorites WHERE user_id = $1 AND object_id = $2 AND object_type = $3;"
	}

	query, err := db.Query(t, id, request.ClubId, Club)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No items found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve", http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}
	defer query.Close()

	//Ответ
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		status bool `json:"new_status"`
	}{
		status: true,
	}
	json.NewEncoder(w).Encode(response)
}

//Поменять статус админки / добавить
//добавить коментарий мб с оценкой
//редактиковать коментарий
//загрузить картинку профиля
//загрузить картинки клуба
//добавить локацию
//получить локации
//загрузить чать отзывов
//получить полную инфу по клубу

//Сделать картинки по адресу (фото профилей и фотоклубов)
//Написать енамы
