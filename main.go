package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/hn275/catapi/internal"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	log := internal.NewLogger()

	db, err := internal.NewDatabase(internal.MustEnv("DATABASE"))
	if err != nil {
		panic(err)
	}

	var totalCat int64
	if err := db.Get(&totalCat, "SELECT COUNT(*) FROM cats;"); err != nil {
		panic(err)
	}

	log.Info(fmt.Sprintf("Randomizing %d cats", totalCat))

	mux := http.NewServeMux()

	mux.Handle("/api/cat", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		catID := rand.Int63n(totalCat)

		var cat internal.CatData
		if err := db.Get(&cat, "SELECT file_type,data FROM cats WHERE id = ?", catID); err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", cat.FileType)
		w.Header().Set("Cache-Control", "no-cache")

		w.WriteHeader(http.StatusOK)

		n, err := w.Write(cat.Data)
		if err != nil {
			log.Error(err.Error())
		} else {
			since := time.Since(now).Microseconds()
			log.Info("served", "user", r.UserAgent(), "ip", r.RemoteAddr, "bytes", n, "time(micro)", since)
		}
	}))

	log.Info("listening on http://127.0.0.1:8080")
	log.Error(http.ListenAndServe(":8080", mux).Error())
}
