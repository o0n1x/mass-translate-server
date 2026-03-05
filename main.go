package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/o0n1x/Sublate/internal/api"
	"github.com/o0n1x/Sublate/internal/database"
	"github.com/o0n1x/sublate-go/provider"
	_ "github.com/o0n1x/sublate-go/provider/deepl"
	"github.com/redis/go-redis/v9"
)

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("SECRET_JWT")
	deeplAPI := os.Getenv("DEEPL_API")
	filepathRoot := "/app/"
	port := "8080"
	db, err := sql.Open("postgres", dbURL)
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

	dbms := database.New(db)
	cfg := api.ApiConfig{}
	cfg.DB = dbms
	cfg.SECRET_JWT = secret
	cfg.Clients = make(map[provider.Provider]provider.Client)
	cfg.Clients[provider.DeepL], err = provider.GetClient(provider.DeepL, deeplAPI)
	if err != nil {
		log.Fatalf("Failed to initalize Deepl Provider: %v", err)
	}
	cfg.Redis = rdb
	cfg.AdminCredentials.Email = os.Getenv("ADMIN_EMAIL")
	cfg.AdminCredentials.Password = os.Getenv("ADMIN_PASSWORD")

	//register admin
	cfg.RegisterAdmin()

	mux := http.NewServeMux()

	mux.Handle(filepathRoot, http.StripPrefix("/app/", http.FileServer(http.Dir("."))))

	mux.HandleFunc("GET /api/health", api.HealthCheck)

	mux.HandleFunc("POST /api/translate", cfg.MiddlewareIsUser(cfg.Translate)) //TODO: provider choice is made in the request not in api path anymore
	mux.HandleFunc("GET /api/documents/{id}", cfg.MiddlewareIsUser(cfg.Status))
	mux.HandleFunc("DELETE /api/documents/{id}", cfg.MiddlewareIsUser(cfg.DeleteDocument))
	mux.HandleFunc("GET /api/documents/{id}/download", cfg.MiddlewareIsUser(cfg.Result))

	mux.HandleFunc("POST /api/auth/login", cfg.Login)
	mux.HandleFunc("POST /api/admin/users", cfg.MiddlewareIsAdmin(cfg.Register))
	mux.HandleFunc("GET /api/admin/users", cfg.MiddlewareIsAdmin(cfg.GetUsers))
	mux.HandleFunc("GET /api/admin/users/{id}", cfg.MiddlewareIsAdmin(cfg.GetUsers))
	mux.HandleFunc("DELETE /api/admin/users/{id}", cfg.MiddlewareIsAdmin(cfg.DeleteUser))
	mux.HandleFunc("PUT /api/admin/users/{id}", cfg.MiddlewareIsAdmin(cfg.UpdateUser))

	s := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(s.ListenAndServe())

}
