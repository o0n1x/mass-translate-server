package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/o0n1x/mass-translate-server/internal/api"
	"github.com/redis/go-redis/v9"
)

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	//secret := os.Getenv("SECRET_JWT")
	deeplAPI := os.Getenv("DEEPL_API")
	filepathRoot := "/app/"
	port := "8080"
	_, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL DB: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	err = rdb.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("Error conntecting to redis : %v", err)
	}

	//dbms := database.New(db)
	cfg := api.ApiConfig{}
	cfg.DeeplClientAPI = deeplAPI
	cfg.Redis = rdb
	mux := http.NewServeMux()

	mux.Handle(filepathRoot, http.StripPrefix("/app/", http.FileServer(http.Dir("."))))

	mux.HandleFunc("GET /api/health", api.HealthCheck)
	mux.HandleFunc("POST /api/deepl/translate", cfg.DeeplTranslate)

	s := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(s.ListenAndServe())

}
