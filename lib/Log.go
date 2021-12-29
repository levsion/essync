package lib

import (
	"log"
	"os"
)

func MyLogFunc() *log.Logger {
	file := "./mylog.txt"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		panic(err)
	}
	return log.New(logFile, "[essync]", log.LstdFlags|log.Lshortfile|log.LUTC)
}
