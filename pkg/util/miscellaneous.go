package util

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
)

func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type Exit struct {
	Err  error
	Code int
}

func ExitAppWithError(code int, err error) {
	panic(Exit{Code: code, Err: err})
}

func OnExit() {
	if r := recover(); r != nil {
		if exit, ok := r.(Exit); ok {
			if exit.Err != nil {
				slog.Error("Exiting with error", "error", exit.Err)
			}
			os.Exit(exit.Code)
		}
		slog.Error("Exiting with panic", "panic", r)
		panic(r)
	}
}
