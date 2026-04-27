package main

import (
	"log"
	"os"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot"
)

func main() {
	if err := releasebot.Run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
