package utils

import (
	"log"
	"os"
	"path/filepath"
)

var ExecutableDir string

func init() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		log.Fatal(err)
	}
	ExecutableDir = filepath.Dir(executable)
}
