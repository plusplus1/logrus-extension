package logrus_extension

import (
	"io/ioutil"
)

import (
	"github.com/plusplus1/logrus-extension/format"
	"github.com/plusplus1/logrus-extension/hook/lvldf"
)

import (
	"github.com/sirupsen/logrus"
)

// InitFileHook, init from config file
func InitFileHook(configYaml string) {

	logrus.StandardLogger().Out = ioutil.Discard     // avoid to print log to screen
	logrus.SetFormatter(format.NewKvTextFormatter()) // set kv formatter
	lvldf.InitLevelDividedFileLogger(configYaml)     // add file log hook

}
