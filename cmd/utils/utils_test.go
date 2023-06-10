package utils_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sath-run/engine/cmd/utils"
)

func TestCompress(t *testing.T) {
	base := "/Users/xinzeng/Downloads"
	filename := filepath.Join(base, "output.tar.gz")
	var buf bytes.Buffer
	if err := utils.Compress(filepath.Join(base, "vina"), &buf); err != nil {
		panic(err)
	}
	os.Remove(filename)
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		panic(err)
	}
	_, err = f.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
	f.Close()
}
