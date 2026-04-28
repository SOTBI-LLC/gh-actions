package main

import (
	"log"
	"os"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
