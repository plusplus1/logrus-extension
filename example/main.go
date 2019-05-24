package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xgo11/stdlog"

	"github.com/juju/fslock"
	"github.com/plusplus1/logrus-extension"
	"github.com/sirupsen/logrus"
)

type sm map[string]interface{}

func (s sm) String() string {
	bs, _ := json.Marshal(s)
	return string(bs)
}

var log = stdlog.Std

func testLog() {
	logrus_extension.EnableDebugLog(true)
	logrus_extension.InitFileHook("test.yaml")
	demoMap := map[string]interface{}{
		"aaa": 1,
	}
	m2 := sm(demoMap)

	fff := func() {

		fields := logrus.Fields{
			"a":     1,
			"b":     "b",
			"e":     errors.New("test error"),
			"c":     "cccccctddd",
			"d":     "tabcd",
			"zh_CN": "中文字符",

			"m2":   m2,
			"demo": demoMap,
		}

		for {
			logrus.WithFields(fields).Info("message")
			logrus.WithFields(fields).Debug("message")
			logrus.WithFields(fields).Warn("message")
			logrus.WithFields(fields).Error("message")

			time.Sleep(1 * time.Second)
		}
	}

	go fff()
	go fff()

}

func testLock() {

	lockFile := filepath.Join(os.TempDir(), "abcde.lock")
	fc := func(id int) {
		for i := 0; i < 100; i++ {
			lc := fslock.New(lockFile)
			if e := lc.Lock(); e == nil {
				log.Infof("g=%d\tlock=ok", id)
				time.Sleep(3 * time.Second)
				if e = lc.Unlock(); e != nil {
					log.Errorf("g=%d\tlock=ok\tunlock=fail\terr=%v", id, e)
				}
			} else {
				log.Errorf("g=%d\tlock=fail\terr=%v", id, e)
			}
		}
	}
	go fc(1)
	go fc(2)
}
func main() {

	//testLock()

	testLog()

	ch := make(chan os.Signal)

	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	<-ch
}
