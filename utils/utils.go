package utils

import (
	"log"
	"os"
	"path/filepath"
)

var SathHome string

func init() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SATH_MODE") == "debug" {
		SathHome = "/tmp/sath"
	} else {
		SathHome = filepath.Dir(executable)
	}
}

func SockFile() string {
	return filepath.Join(SathHome, "sath.sock")
}
