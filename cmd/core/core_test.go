package core_test

import (
	"fmt"
)

func checkErr(err error) {
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
}
