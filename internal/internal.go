package internal

import (
	"fmt"
	"log/slog"
	"os"
)

var log *slog.Logger

func init() {
	log = slog.Default()
}

func MustEnv(key string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		panic(fmt.Sprintf("key %s not set", key))
	}
	return val
}

func NewLogger() *slog.Logger {
	return log
}
