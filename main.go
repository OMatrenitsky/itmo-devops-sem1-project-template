package main

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

const (
	dbUser     = "validator"
	dbPassword = "val1dat0r"
	dbName     = "project-sem-1"
	dbHost     = "localhost"
	dbPort     = "5432"
)

var db *sql.DB

type PostResponse struct {
	TotalItems      int     `json:"total_items"`
	TotalCategories int     `json:"total_categories"`
	TotalPrice      float64 `json:"total_price"`
}

// Инициализация базы данных
func initDB() {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}

// Обработчик для POST /api/v0/prices
func handlePostPrices(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	zipReader, err := zip.NewReader(file.(io.ReaderAt), r.ContentLength)
	if err != nil {
		http.Error(w, "invalid zip", http.StatusBadRequest)
		return
	}

	tx, _ := db.Begin()
	defer tx.Rollback()

	for _, f := range zipReader.File {
		if f.Name != "data.csv" {
			continue
		}

		csvFile, _ := f.Open()
		defer csvFile.Close()

		reader := csv.NewReader(csvFile)

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}

			price, _ := strconv.ParseFloat(record[4], 64)

			tx.Exec(`
				INSERT INTO prices (id, created_at, name, category, price)
				VALUES ($1, $2, $3, $4, $5)
			`, record[0], record[1], record[2], record[3], price)
		}
	}

	var resp PostResponse
	tx.QueryRow(`SELECT COUNT(*) FROM prices`).Scan(&resp.TotalItems)
	tx.QueryRow(`SELECT COUNT(DISTINCT category) FROM prices`).Scan(&resp.TotalCategories)
	tx.QueryRow(`SELECT COALESCE(SUM(price),0) FROM prices`).Scan(&resp.TotalPrice)

	tx.Commit()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Обработчик для GET /api/v0/prices
func handleGetPrices(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`SELECT id, created_at, name, category, price FROM prices`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="data.zip"`)

	zipWriter := zip.NewWriter(w)
	csvFile, _ := zipWriter.Create("data.csv")
	writer := csv.NewWriter(csvFile)

	for rows.Next() {
		var id, created, name, category string
		var price float64
		rows.Scan(&id, &created, &name, &category, &price)

		writer.Write([]string{
			id, created, name, category, fmt.Sprintf("%.2f", price),
		})
	}

	writer.Flush()
	zipWriter.Close()
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/api/v0/prices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlePostPrices(w, r)
			return
		}
		if r.Method == http.MethodGet {
			handleGetPrices(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
