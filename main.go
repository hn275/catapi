package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/hn275/catapi/internal"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

type App struct {
	totalCat int64
	db       *sqlx.DB
	logger   *slog.Logger
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	logger := internal.NewLogger()

	db, err := internal.NewDatabase(internal.MustEnv("DATABASE"))
	if err != nil {
		panic(err)
	}

	var totalCat int64
	if err := db.Get(&totalCat, "SELECT COUNT(*) FROM cats;"); err != nil {
		panic(err)
	}

	logger.Info(fmt.Sprintf("Randomizing %d cats", totalCat))

	app := App{totalCat, db, logger}
	mux := http.NewServeMux()

	mux.Handle("/", serve(&app))

	logger.Info("listening on http://127.0.0.1:8080")
	logger.Error(http.ListenAndServe(":8080", mux).Error())
}

func serve(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		catID := rand.Int63n(app.totalCat)

		q := "SELECT file_type,data FROM cats WHERE id = ?"
		var cat internal.CatData
		if err := app.db.Get(&cat, q, catID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			since := time.Since(now).Microseconds()
			app.logger.Error("error", err.Error(), "user", r.UserAgent(), "ip", r.RemoteAddr, "time(micro)", since)
			return
		}

		w.Header().Set("Content-Type", cat.FileType)
		w.Header().Set("Cache-Control", "no-cache")

		w.WriteHeader(http.StatusOK)

		n, err := w.Write(cat.Data)
		since := time.Since(now).Microseconds()
		if err != nil {
			app.logger.Error("error", err.Error(), "user", r.UserAgent(), "ip", r.RemoteAddr, "time(micro)", since)
		} else {
			app.logger.Info("served", "user", r.UserAgent(), "ip", r.RemoteAddr, "bytes", n, "time(micro)", since)
		}
	}
}
