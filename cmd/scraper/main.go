package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/hn275/catapi/internal"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

const (
	baseURL string = "https://api.thecatapi.com/v1"
	listURL string = "https://api.thecatapi.com/v1/images/search?limit=10"
)

type Cats []CatResponse

type CatResponse struct {
	Breeds     []interface{} `json:"breeds"`
	ID         string        `json:"id"`
	URL        string        `json:"url"`
	Width      int64         `json:"width"`
	Height     int64         `json:"height"`
	Categories []Category    `json:"categories,omitempty"`
}

type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func main() {
	log := internal.NewLogger()

	if err := godotenv.Load(); err != nil {
		log.Error(err.Error())
	}

	db, err := internal.NewDatabase(internal.MustEnv("DATABASE"))
	if err != nil {
		panic(err)
	}

	client := http.Client{}
	apiToken := internal.MustEnv("API_KEY")

	wg := new(sync.WaitGroup)
	mtx := new(sync.Mutex)

	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: false,
	})

	if err != nil {
		panic(err)
	}

	for i := 0; i < 20; i++ {
		url := makeUrl("/images/search", map[string]string{
			"page":  fmt.Sprintf("%d", i),
			"limit": "10",
		})

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			panic(err)
		}

		req.Header.Add("x-api-key", apiToken)

		log.Info("metadata", "page", i)
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		var cats Cats
		if err := json.NewDecoder(res.Body).Decode(&cats); err != nil {
			panic(err)
		}

		for _, cat := range cats {
			log.Info("queued", "catid", cat.ID, "url", cat.URL)
			wg.Add(1)
			go fetchImage(cat, wg, mtx, tx)
		}
	}

	wg.Wait()

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func fetchImage(cat CatResponse, wg *sync.WaitGroup, mtx *sync.Mutex, tx *sqlx.Tx) {
	defer wg.Done()
	log := internal.NewLogger()

	client := http.Client{}

	res, err := client.Get(cat.URL)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	responseReader := bufio.NewReader(res.Body)

	extension := cat.URL[len(cat.URL)-3:]

	buf := bytes.Buffer{}
	n, err := responseReader.WriteTo(&buf)

	if err != nil {
		log.Error("data copied failed???")
		return
	}

	if res.ContentLength != n {
		log.Error("bytes copied failed???")
		return
	}

	log.Info("downloaded", "catid", cat.ID, "url", cat.URL, "size", buf.Len())

	catData := internal.CatData{
		CatID:    cat.ID,
		FileType: fmt.Sprintf("image/%s", extension),
		Data:     buf.Bytes(),
	}

	mtx.Lock()
	defer mtx.Unlock()

	_, err = tx.NamedExec(
		`INSERT INTO cats(cat_id, data, file_type)
            VALUES (:cat_id, :data, :file_type);`,
		catData)

	if err != nil {
		log.Error("failed to write cat to db", "catid", cat.ID, "err", err)
	} else {
		log.Info("saved", "id", catData.CatID, "ext", catData.FileType)
	}
}

func makeUrl(path string, query map[string]string) string {
	params := url.Values{}
	for k, v := range query {
		params.Add(k, v)
	}

	url, err := url.JoinPath(baseURL, path)
	if err != nil {
		panic(err)
	}

	return url + "?" + params.Encode()
}
