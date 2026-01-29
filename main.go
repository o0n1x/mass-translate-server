package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/o0n1x/mass-translate-server/internal/api"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	//secret := os.Getenv("SECRET_JWT")
	//deeplAPI := os.Getenv("DEEPL_API")
	filepathRoot := "/app/"
	port := "8080"
	_, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}

	//dbms := database.New(db)
	cfg := api.ApiConfig{}
	cfg.DeeplClientAPI = os.Getenv("DEEPL_API")
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
