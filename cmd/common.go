package cmd

import (
	"log"
	"os"
)

var (
	BuildVersion = "dev"
	BuildTime    = "unknown"
	BuildHash    = "unknown"
	BuildUser    = "unknown"
)

func Version(logger *log.Logger, app string) {
	logger.Printf("%s version %s built %s\n", app, BuildVersion, BuildTime)
	logger.Printf("commit: %s by %s\n", BuildHash, BuildUser)
	os.Exit(0)
}

func Help(logger *log.Logger, msg string) {
	logger.Println(msg)
	os.Exit(1)
}
