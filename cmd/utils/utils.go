package utils

import (
	"fmt"
	"time"
)

func LogError(err error) {
	fmt.Printf(
		"[GIN] %v |%+v\n",
		time.Now().Format("2006/01/02 - 15:04:05"),
		err)
}
