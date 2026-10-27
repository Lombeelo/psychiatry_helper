package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/backend/appservice"
	"example.com/backend/auth"
	"example.com/backend/httpapi"
	"example.com/backend/seed"
	sqlitestore "example.com/backend/sqlite"
	"github.com/joho/godotenv"

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден; используются переменные окружения системы")
	}

	addr := env("HTTP_ADDR", ":8080")
	dsn := env("SQLITE_DSN", "./data/app.db")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := sqlitestore.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer store.Close()

	credentials, err := seed.EnsureDemoData(ctx, store)
	if err != nil {
		return err
	}

	service := appservice.NewSQLiteService(store)

	secret := env("JWT_SECRET", "dev-only-change-me")
	api := httpapi.New(auth.NewJWT(secret), service)

	log.Println("============================================================")
	log.Println("Тестовые учётные записи для входа:")
	log.Printf("  врач:    логин %q, пароль %q", credentials.DoctorLogin, credentials.DoctorPassword)
	log.Printf("  пациент: логин %q, пароль %q", credentials.PatientLogin, credentials.PatientPassword)
	log.Println("============================================================")

	server := &http.Server{
		Addr:              addr,
		Handler:           api,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on %s", addr)
		serverErr <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err

	case <-signalCtx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer shutdownCancel()

		return server.Shutdown(shutdownCtx)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
