package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	//do your serializing here
	return []byte(strconv.Itoa(time.Time(t).Nanosecond())), nil
}

var jobLogger *lumberjack.Logger
var errLogger *lumberjack.Logger

func init() {
	dir, err := GetExecutableDir()
	if err != nil {
		log.Fatal(err)
	}
	loggerDir := filepath.Join(dir, "log")
	err = os.MkdirAll(loggerDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	jobLogger = &lumberjack.Logger{
		Filename:   filepath.Join(loggerDir, "jobs.log"),
		MaxSize:    1,     // megabytes
		MaxBackups: 3,     //
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
	errLogger = &lumberjack.Logger{
		Filename:   filepath.Join(loggerDir, "jobs.log"),
		MaxSize:    20,    // megabytes
		MaxBackups: 3,     //
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
}

func GetExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executable, err = filepath.EvalSymlinks(executable)
	dir := filepath.Dir(executable)
	return dir, err
}

func LogError(err error) {
	if os.Getenv("SATH_MODE") == "debug" {
		msg := fmt.Sprintf(
			"[SATH Err] %v |%+v\n",
			time.Now().Format("2006/01/02 - 15:04:05"),
			err)
		log.Print(msg)
		errLogger.Write([]byte(msg))
	}
}

func LogDebug(a ...any) {
	if os.Getenv("SATH_MODE") == "debug" {
		messages := make([]interface{}, 0)
		messages = append(messages,
			"[SATH DEBUG] ",
			time.Now().Format("2006/01/02 - 15:04:05"),
			" | ")
		messages = append(messages, a...)
		fmt.Println(messages...)
	}
}

func LogWarning(a ...any) {
	messages := make([]interface{}, 0)
	messages = append(messages,
		"[SATH Warning] ",
		time.Now().Format("2006/01/02 - 15:04:05"),
		" | ")
	messages = append(messages, a...)
	fmt.Println(messages...)
}

func LogJob(content []byte) {
	jobLogger.Write(content)
	jobLogger.Write([]byte("\n"))
}
