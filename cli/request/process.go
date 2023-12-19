package request

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func FindRunningDaemonPid() (pid int, err error) {
	os.Getpid()
	command := exec.Command("bash", "-c", "ps | grep sath-engine | grep -v grep | grep -v \"go run\"")
	res, err := command.Output()
	if err != nil {
		err = errors.New("cannot find the running pid of sath")
		return
	}
	pid, err = strconv.Atoi(strings.Fields(string(res))[0])
	if err != nil {
		return
	}
	return
}
