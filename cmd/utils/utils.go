package utils

import (
	"log"
	"time"
)

func LogError(err error) {
	log.Printf(
		"[SATH] %v |%+v\n",
		time.Now().Format("2006/01/02 - 15:04:05"),
		err)
}
