package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

var tmpl = template.Must(template.ParseFiles("template.html"))

type Player struct {
	PlayerID   string
	Jersey     int
	Fname      string
	Sname      string
	Position   string
	Birthday   string
	Weight     int
	Height     int
	BirthCity  string
	BirthState string
}

func dbConn() (*sql.DB, error) {
	connStr := "user=postgres password=666666 dbname=players sslmode=disable"
	return sql.Open("postgres", connStr)
}

func home(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
}

func getPlayers(w http.ResponseWriter, r *http.Request) {
	db, err := dbConn()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	position := r.FormValue("position")
	birthYearFrom := r.FormValue("birthYearFrom")
	birthYearTo := r.FormValue("birthYearTo")

	query := "SELECT playerid, jersey, fname, sname, position, birthday, weight, height, birthcity, birthstate FROM players WHERE 1=1"
	var args []interface{}

	if position != "" {
		query += " AND position = $1"
		args = append(args, position)
	}
	if birthYearFrom != "" {
		query += " AND EXTRACT(YEAR FROM birthday) >= $" + strconv.Itoa(len(args)+1)
		args = append(args, birthYearFrom)
	}
	if birthYearTo != "" {
		query += " AND EXTRACT(YEAR FROM birthday) <= $" + strconv.Itoa(len(args)+1)
		args = append(args, birthYearTo)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		var p Player
		err := rows.Scan(&p.PlayerID, &p.Jersey, &p.Fname, &p.Sname, &p.Position, &p.Birthday, &p.Weight, &p.Height, &p.BirthCity, &p.BirthState)
		if err != nil {
			log.Fatal(err)
		}
		players = append(players, p)
	}

	if err := tmpl.Execute(w, players); err != nil {
		log.Fatal(err)
	}
}

type UpdateRequest struct {
	PlayerID string `json:"playerId"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

func updatePlayer(w http.ResponseWriter, r *http.Request) {
	var updateReq UpdateRequest
	err := json.NewDecoder(r.Body).Decode(&updateReq)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db, err := dbConn()
	if err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Validate new value based on original filter criteria.
	if updateReq.Field == "birthday" {
		year, err := strconv.Atoi(updateReq.Value[:4]) // Extract the year from the date
		if err != nil || year < 1900 || year > 2024 {
			json.NewEncoder(w).Encode(map[string]bool{"success": false})
			return
		}
	}

	var query string
	switch updateReq.Field {
	case "birthday":
		query = "UPDATE players SET birthday = $1 WHERE playerid = $2"
	case "birthstate":
		query = "UPDATE players SET birthstate = $1 WHERE playerid = $2"
	default:
		http.Error(w, "Invalid field", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(query, updateReq.Value, updateReq.PlayerID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]bool{"success": false})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/players", getPlayers)
	http.HandleFunc("/update-player", updatePlayer)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
