package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

type Entry = logrus.Entry

func Init() {
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	Log.SetOutput(os.Stdout)

	if os.Getenv("DEBUG") == "true" {
		Log.SetLevel(logrus.DebugLevel)
	} else {
		Log.SetLevel(logrus.InfoLevel)
	}
}
