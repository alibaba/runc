package utils

import (
	"os"

	"github.com/sirupsen/logrus"
)

const defaultLogPath = "/run/runc.log"

var(
	logger = NewLogger()
)

func GetLogger() *logrus.Logger {
	return logger
}

func NewLogger() *logrus.Logger {
	f,err := os.OpenFile(defaultLogPath, os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0x644)
	if err != nil{
		panic(err)
	}

	return &logrus.Logger{
		Out:       f,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}
}