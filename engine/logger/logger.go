package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sath-run/engine/utils"
	"gopkg.in/natefinch/lumberjack.v2"
)

var jobLogger *lumberjack.Logger
var errLogger *lumberjack.Logger
var stdLogger *lumberjack.Logger

func Init() error {
	loggerDir := filepath.Join(utils.ExecutableDir, "/log")
	if err := os.MkdirAll(loggerDir, os.ModePerm); err != nil {
		return err
	}
	jobLogger = &lumberjack.Logger{
		Filename:   filepath.Join(loggerDir, "jobs.log"),
		MaxSize:    1,     // megabytes
		MaxBackups: 3,     //
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
	errLogger = &lumberjack.Logger{
		Filename:   filepath.Join(loggerDir, "err.log"),
		MaxSize:    20,    // megabytes
		MaxBackups: 3,     //
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
	stdLogger = &lumberjack.Logger{
		Filename:   filepath.Join(loggerDir, "out.log"),
		MaxSize:    20,    // megabytes
		MaxBackups: 3,     //
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
	return nil
}

func Error(err error) {
	if os.Getenv("SATH_MODE") == "debug" {
		msg := fmt.Sprintf(
			"[SATH Err] %v |%+v\n",
			time.Now().Format("2006/01/02 - 15:04:05"),
			err)
		log.Print(msg)
		errLogger.Write([]byte(msg))
	}
}

func Debug(a ...any) {
	if os.Getenv("SATH_MODE") == "debug" {
		messages := make([]interface{}, 0)
		messages = append(messages,
			"[SATH DEBUG] ",
			time.Now().Format("2006/01/02 - 15:04:05"),
			" | ")
		messages = append(messages, a...)
		stdLogger.Write([]byte(fmt.Sprintln(messages...)))
	}
}

func Warning(a ...any) {
	messages := make([]interface{}, 0)
	messages = append(messages,
		"[SATH Warning] ",
		time.Now().Format("2006/01/02 - 15:04:05"),
		" | ")
	messages = append(messages, a...)
	stdLogger.Write([]byte(fmt.Sprintln(messages...)))
}

func LogJob(content []byte) {
	jobLogger.Write(content)
	jobLogger.Write([]byte("\n"))
}
