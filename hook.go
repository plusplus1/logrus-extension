package logrus_extension

import (
	"os"
	"strings"
)
import (
	"github.com/sirupsen/logrus"
)

// levelFilteredTimeRotatedHook: every single file for each maxLevel of log message
type levelFilteredTimeRotatedHook struct {
	logName      string                         // log name , with maxLevel suffix
	logDir       string                         // log directory
	maxLevel     logrus.Level                   // max maxLevel
	allLevels    []logrus.Level                 // all supported levels
	writers      map[logrus.Level][]entryWriter // writers, do real work
	rotateEnable bool                           // log split related fields
	rotateBy     string
	bufferEnable bool // buf enable
}

var (
	flagEnableDebug = false
)

func init() {
	EnableDebugLog(false)
}

func EnableDebugLog(b bool) {
	flagEnableDebug = b
}

func debugF(f string, args ...interface{}) {
	if flagEnableDebug {
		log.Debugf(f, args...)
	}
}

func newLevelFilteredTimeRotatedHook(conf lvlConf) logrus.Hook {

	var err error
	aHook := &levelFilteredTimeRotatedHook{}
	aHook.logDir = conf.Directory
	aHook.logName = conf.Filename
	aHook.maxLevel, err = logrus.ParseLevel(conf.Level)

	if err != nil {
		log.Panicf("init logger hook failed, %v", err)
	}

	if len(aHook.Levels()) < 1 {
		log.Panicf("init logger hook failed, invalid level:%v", conf.Level)
	}

	// replace hostname in filename
	if strings.Index(aHook.logName, "${hostname}") > 0 {
		host, _ := os.Hostname()
		aHook.logName = strings.Replace(aHook.logName, "${hostname}", host, 1)
	}

	// ensure directory exists
	_ = os.MkdirAll(aHook.logDir, 0775)
	if info, err := os.Stat(aHook.logDir); err != nil || !info.IsDir() {
		log.Panicf("init logger hook failed, %v", err)
	}

	// init time rotated writers
	aHook.writers = map[logrus.Level][]entryWriter{}
	for _, level := range aHook.Levels() {
		aHook.writers[level] = []entryWriter{
			newRotateWriter(level, aHook.logDir, aHook.logName, conf.Rotate),
		}
	}
	return aHook

}

func (hook *levelFilteredTimeRotatedHook) Levels() []logrus.Level {
	if len(hook.allLevels) < 1 {
		for _, lvl := range logrus.AllLevels {
			if lvl <= hook.maxLevel && lvl >= lvlMinLevel {
				hook.allLevels = append(hook.allLevels, lvl)
			}
		}
	}
	return hook.allLevels
}

func (hook *levelFilteredTimeRotatedHook) Fire(entry *logrus.Entry) (err error) {

	if wLst, ok := hook.writers[entry.Level]; ok {
		var bytes []byte
		if bytes, err = entry.Logger.Formatter.Format(entry); err != nil {
			log.Errorf(" *** Format entry failed, %v", err)
			return err
		}

		for _, w := range wLst {
			if err = w.WriteEntry(entry, bytes); err != nil {
				log.Errorf(" *** WriteEntry to %v failed, %v", w, err)
			}
		}
	}
	return
}
