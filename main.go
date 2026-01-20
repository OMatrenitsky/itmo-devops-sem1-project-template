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
	"time"

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

	type PriceRow struct {
		Name       string
		Category   string
		Price      float64
		CreateDate time.Time
	}

	var rows []PriceRow

	// Полностью читаем CSV ДО транзакции
	for _, f := range zipReader.File {
		if f.Name != "data.csv" {
			continue
		}

		csvFile, err := f.Open()
		if err != nil {
			http.Error(w, "cannot open csv", http.StatusBadRequest)
			return
		}
		defer csvFile.Close()

		reader := csv.NewReader(csvFile)

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, "csv read error", http.StatusBadRequest)
				return
			}

			createDate, err := time.Parse("2006-01-02", record[1])
			if err != nil {
				http.Error(w, "invalid date format", http.StatusBadRequest)
				return
			}

			price, err := strconv.ParseFloat(record[4], 64)
			if err != nil {
				http.Error(w, "invalid price", http.StatusBadRequest)
				return
			}

			rows = append(rows, PriceRow{
				Name:       record[2],
				Category:   record[3],
				Price:      price,
				CreateDate: createDate,
			})
		}
	}

	// Открываем транзакцию только после чтения CSV
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO prices (name, category, price, create_date)
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	for _, r := range rows {
		_, err := stmt.Exec(
			r.Name,
			r.Category,
			r.Price,
			r.CreateDate,
		)
		if err != nil {
			http.Error(w, "insert error", http.StatusInternalServerError)
			return
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		http.Error(w, "commit error", http.StatusInternalServerError)
		return
	}

	var resp PostResponse

	// Количество элементов текущей загрузки
	resp.TotalItems = len(rows)

	// Статистика по всей базе
	err = db.QueryRow(`
	SELECT
		COUNT(DISTINCT category),
		COALESCE(SUM(price), 0)
	FROM prices
`).Scan(
		&resp.TotalCategories,
		&resp.TotalPrice,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Обработчик для GET /api/v0/prices
func handleGetPrices(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`
		SELECT id, create_date, name, category, price
		FROM prices
		ORDER BY id
	`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PriceRow struct {
		ID         int
		CreateDate time.Time
		Name       string
		Category   string
		Price      float64
	}

	var data []PriceRow

	// Полностью вычитываем курсор
	for rows.Next() {
		var r PriceRow
		if err := rows.Scan(
			&r.ID,
			&r.CreateDate,
			&r.Name,
			&r.Category,
			&r.Price,
		); err != nil {
			http.Error(w, "db scan error", http.StatusInternalServerError)
			return
		}
		data = append(data, r)
	}

	// Проверяем ошибку курсора
	if err := rows.Err(); err != nil {
		http.Error(w, "db rows error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="data.zip"`)

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	csvFile, err := zipWriter.Create("data.csv")
	if err != nil {
		http.Error(w, "zip create error", http.StatusInternalServerError)
		return
	}

	writer := csv.NewWriter(csvFile)

	for _, r := range data {
		err := writer.Write([]string{
			strconv.Itoa(r.ID),
			r.CreateDate.Format("2006-01-02"),
			r.Name,
			r.Category,
			fmt.Sprintf("%.2f", r.Price),
		})
		if err != nil {
			http.Error(w, "csv write error", http.StatusInternalServerError)
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, "csv flush error", http.StatusInternalServerError)
		return
	}
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
