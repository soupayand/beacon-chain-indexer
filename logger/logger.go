package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

var errorLogger *log.Logger
var infoLogger *log.Logger

func InitLogger() {
	file, err := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	errorLogger = log.New(file, "", log.LstdFlags)

	file, err = os.OpenFile("info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	infoLogger = log.New(file, "", log.LstdFlags)

}

func LogError(err error) {
	if errorLogger != nil {
		_, file, line, _ := runtime.Caller(1)
		filename := filepath.Base(file)
		errorLogger.Output(2, fmt.Sprintf("[ERROR] %s:%d %v", filename, line, err.Error()))
	}
}

func LogInfo(v ...any) {
	if infoLogger != nil {
		_, file, line, _ := runtime.Caller(1)
		filename := filepath.Base(file)
		infoLogger.Output(2, fmt.Sprintf("[INFO] %s:%d %v", filename, line, v))
	}
}
