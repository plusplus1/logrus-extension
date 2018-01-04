package main

import (
	"errors"
	"time"

	"github.com/plusplus1/logrus-extension"
	"github.com/sirupsen/logrus"
)

func main() {

	logrus_extension.InitFileHook("test.yaml")

	for {

		fields := logrus.Fields{
			"a":     1,
			"b":     "b",
			"e":     errors.New("test error"),
			"c":     "ccc\nccc\tddd",
			"d":     "\tabcd",
			"zh_CN": "中文字符",
		}

		logrus.WithFields(fields).Info("message")
		logrus.WithFields(fields).Debug("message")
		logrus.WithFields(fields).Warn("message")
		logrus.WithFields(fields).Error("message")

		time.Sleep(100 * time.Millisecond)
	}
}
