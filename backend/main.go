package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	DbName     string
	DbHost     string
	DbPort     int
	DbUsername string
	DbPassword string
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", HealthCheck)
	return mux
}

func loadConfig() (*DatabaseConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		return nil, err
	}
	config := DatabaseConfig{
		DbName:     os.Getenv("DB_NAME"),
		DbHost:     os.Getenv("DB_HOST"),
		DbPassword: os.Getenv("DB_PASSWORD"),
		DbPort:     port,
		DbUsername: os.Getenv("DB_USERNAME"),
	}
	return &config, nil
}
func run(ctx context.Context) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	conn, err := pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", config.DbUsername, config.DbPassword, config.DbHost, config.DbPort, config.DbName))
	if err != nil {
		return err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)
	srv := http.Server{
		Addr:    ":8080",
		Handler: routes(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()
	select {
	case <-ctx.Done():
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctxShutdown); err != nil {
			return err
		}
	case err := <-errChan:
		return err
	}
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
