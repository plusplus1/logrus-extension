package logrus_extension

/**

logrus hook，按照日志级别分别记录到对应的文件种
-------
1、按日志级别输出；
2、支持按天切分日志；
3、自动清理过期日志（30天）；

*/

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/go-yaml/yaml"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	lvlMinLevel    = logrus.ErrorLevel
	lvlRotateByDay = "D"
)

var (
	lvlSuffixMapping = map[logrus.Level]string{
		logrus.DebugLevel: logrus.DebugLevel.String(),
		logrus.InfoLevel:  logrus.InfoLevel.String(),
		logrus.WarnLevel:  logrus.WarnLevel.String()[0:4],
		logrus.ErrorLevel: logrus.ErrorLevel.String(),
	}
)

const (
	lvlDefaultLogBufferSize = 1024 * 10
)

type (
	lvlConf struct {
		Directory string `yaml:"directory"`
		Filename  string `yaml:"filename"`
		Level     string `yaml:"level"`
		Daily     bool   `yaml:"daily"`
		Buffer    bool   `yaml:"buffer"`
	}

	// lvlHook: every single file for each level of log message
	lvlHook struct {
		logName      string                           // log name , with level suffix
		logDir       string                           // log directory
		level        logrus.Level                     // min level
		allLevels    []logrus.Level                   // all supported levels
		writers      map[logrus.Level]*lvlMutexWriter // writers, do real work
		rotateEnable bool                             // log split related fields
		rotateBy     string
		bufferEnable bool // buf enable
	}

	lvlMutexWriter struct {
		sync.Mutex

		level          logrus.Level  // level
		hook           *lvlHook      // hook
		dstFd          *os.File      // log output file fd
		dstFile        string        // log output file name
		bufMutex       *sync.Mutex   // buffer mutex lock
		bufWriter      *bufio.Writer // buffer writer
		lockFd         int           // file lock fd
		lockFile       string        // file lock file name
		lastRotateTime time.Time     // last rotate time
	}
)

func NewLvlConf(yamlFile string) (conf lvlConf, err error) {

	var bytes []byte

	if bytes, err = ioutil.ReadFile(yamlFile); err == nil {
		conf = lvlConf{Level: logrus.InfoLevel.String(), Daily: true, Buffer: false}
		if err = yaml.Unmarshal(bytes, &conf); err == nil {
			conf.Level = strings.ToLower(conf.Level)
			return
		}
	}
	if err == nil {
		err = errors.New("load " + yamlFile + " fail")
	}
	return
}

func (hook *lvlHook) Levels() []logrus.Level {
	if len(hook.allLevels) < 1 {
		for _, lvl := range logrus.AllLevels {
			if lvl <= hook.level && lvl >= lvlMinLevel {
				hook.allLevels = append(hook.allLevels, lvl)
			}
		}
	}
	return hook.allLevels
}

func (hook *lvlHook) Fire(entry *logrus.Entry) (err error) {
	if w, ok := hook.writers[entry.Level]; ok {
		var msg string
		if msg, err = entry.String(); err != nil {
			return err
		}
		err = w.writeMessage(entry.Level, entry.Time, msg)
	}
	return
}

func registerLvlHookWithLogger(configYaml string, logger *logrus.Logger) {
	aHook := &lvlHook{writers: make(map[logrus.Level]*lvlMutexWriter)}

	if conf, e := NewLvlConf(configYaml); e == nil {

		os.MkdirAll(conf.Directory, 0775)
		if info, err := os.Stat(conf.Directory); err != nil || !info.IsDir() {
			log.Panicf("make log directory fail, error=%v", err)
		}
		aHook.logDir = conf.Directory

		if conf.Filename == "" {
			log.Panicf("conf [%s] invalid, filename empty", configYaml)
		}
		aHook.logName = conf.Filename
		if strings.Index(aHook.logName, "${hostname}") > 0 {
			osHostName, _ := os.Hostname()
			aHook.logName = strings.Replace(aHook.logName, "${hostname}", osHostName, 1)
		}

		if aHook.level, e = logrus.ParseLevel(conf.Level); e != nil {
			log.Panicf("conf [%s] invalid, level parse fail", configYaml)
		}

		if logger != nil {
			logger.SetLevel(aHook.level)
		} else {
			logrus.SetLevel(aHook.level)
		}

		if len(aHook.Levels()) < 1 {
			log.Panicf("no valid levels for hook")
		}

		if conf.Daily {
			aHook.rotateEnable = true
			aHook.rotateBy = lvlRotateByDay
		}

		if conf.Buffer {
			aHook.bufferEnable = true
		}
	} else {
		log.Panicf("parse config [%s] fail, error=%v", configYaml, e)
	}

	for _, lvl := range aHook.Levels() {
		aHook.writers[lvl] = initLvlMutexWriter(lvl, aHook)
	}

	if logger == nil {
		logrus.AddHook(aHook)
	} else {
		logger.AddHook(aHook)
	}
}

func initLvlMutexWriter(ll logrus.Level, hk *lvlHook) *lvlMutexWriter {
	writer := &lvlMutexWriter{
		dstFile:  filepath.Join(hk.logDir, fmt.Sprintf("%s.%s", hk.logName, lvlSuffixMapping[ll])),
		level:    ll,
		hook:     hk,
		bufMutex: &sync.Mutex{},
	}

	if writer.hook.bufferEnable {
		go writer.flushEverySecond()
	}
	return writer
}

func (writer *lvlMutexWriter) writeMessage(level logrus.Level, logTime time.Time, message string) error {

	if writer.hook.rotateEnable {
		writer.checkAndDoRotate(logTime)
	}

	if writer.hook.bufferEnable && writer.bufWriter != nil {
		writer.bufMutex.Lock()
		_, err := writer.bufWriter.WriteString(message)
		writer.bufMutex.Unlock()
		return err
	} else if writer.dstFd != nil {
		_, err := writer.dstFd.WriteString(message)
		return err
	}

	return nil
}

func (writer *lvlMutexWriter) flushEverySecond() {
	t := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-t.C:
			writer.flushOnce()
		}
	}
}

func (writer *lvlMutexWriter) flushOnce() {

	if writer.bufWriter != nil {
		writer.bufMutex.Lock()
		writer.bufWriter.Flush()
		writer.bufMutex.Unlock()
	}
}

func (writer *lvlMutexWriter) startLogWriter() error {

	if writer.lockFd == 0 {
		lockSign := fmt.Sprintf("%x", md5.Sum([]byte(writer.dstFile)))
		parts := strings.Split(writer.dstFile, "/")

		writer.lockFile = filepath.Join(os.TempDir(), parts[len(parts)-1]+"."+lockSign+".lock")
		writer.lockFd, _ = unix.Open(writer.lockFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	}

	if fd, err := writer.createLogFile(); err != nil {
		return err
	} else {
		writer.setFd(fd)
		return nil
	}
}

func (writer *lvlMutexWriter) createLogFile() (*os.File, error) {
	return os.OpenFile(writer.dstFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0665)
}

func (writer *lvlMutexWriter) setFd(fd *os.File) {
	if fd == nil {
		return
	}

	if writer.dstFd != nil {
		writer.flush()
		writer.destroy()
	}

	writer.dstFd = fd

	if writer.hook.bufferEnable {
		if writer.bufWriter == nil {
			writer.bufWriter = bufio.NewWriterSize(writer.dstFd, lvlDefaultLogBufferSize)
		} else {
			writer.bufWriter.Reset(writer.dstFd)
		}
	}

	if fdInfo, _ := writer.dstFd.Stat(); fdInfo != nil {
		writer.lastRotateTime = fdInfo.ModTime()
	} else {
		writer.lastRotateTime = time.Now()
	}
}

func (writer *lvlMutexWriter) checkShouldRotate(logTime time.Time) bool {
	if !writer.hook.rotateEnable {
		return false
	}

	switch writer.hook.rotateBy {
	case lvlRotateByDay:
		/*

			if writer.lastRotateTime.Minute() != logTime.Minute() {
				return true
			}
		*/
		if writer.lastRotateTime.Day() != logTime.Day() {
			return true
		}
	}

	if writer.lastRotateTime.Month() != logTime.Month() {
		return true
	}
	if writer.lastRotateTime.Year() != logTime.Year() {
		return true
	}
	return false

}

func (writer *lvlMutexWriter) checkAndDoRotate(logTime time.Time) error {
	writer.Lock()
	defer writer.Unlock()

	if writer.dstFd == nil {
		writer.startLogWriter()
		return nil
	}

	if !writer.checkShouldRotate(logTime) {
		return nil
	}

	writer.flush()
	writer.destroy()

	var err error

	doBackup := func() error {
		_, e1 := os.Lstat(writer.dstFile)
		backupFile := writer.dstFile + "." + writer.lastRotateTime.Format("2006-01-02")

		_, e2 := os.Lstat(backupFile)

		if e1 == nil && e2 != nil {
			if err = os.Rename(writer.dstFile, backupFile); err != nil {
				log.Printf("[ERROR] split log error, rename fail, %s", err.Error())
			}
			return err
		}

		if e1 == nil && e2 == nil {
			log.Printf("[WARN] %v may be has backuped to %v", writer.dstFile, backupFile)
			return nil
		}

		if e1 != nil && e2 != nil {
			log.Printf("[WARN] split log fail, both file not exist, %s", backupFile)
			return errors.New("both file not exists")
		}

		if e1 != nil && e2 == nil {
			log.Printf("[INFO] split log may be has done, %s", backupFile)
			return nil
		}

		return nil
	}

	if e := accuireFileLock(uintptr(writer.lockFd)); e == nil {
		err = doBackup()
		releaseFileLock(uintptr(writer.lockFd))
	} else {
		log.Printf("[ERROR] do log rotate fail, flock error=%v", e)
	}

	if err != nil {
		log.Printf("[ERROR] daily rotate log %v error=%v", writer.dstFile, err)
	}

	err = writer.startLogWriter()

	go writer.deleteOldLog()
	return err
}

func (writer *lvlMutexWriter) destroy() {
	if writer.dstFd != nil {
		writer.dstFd.Close()
		writer.dstFd = nil
	}
}

func (writer *lvlMutexWriter) flush() {
	if writer.dstFd != nil {
		if writer.hook.bufferEnable {
			if writer.bufWriter != nil {
				writer.bufMutex.Lock()
				writer.bufWriter.Flush()
				writer.bufMutex.Unlock()
			}
		} else {
			writer.dstFd.Sync()
		}
	}
}

func (writer *lvlMutexWriter) deleteOldLog() {

	nowTime := time.Now()
	oneDay := 24 * time.Hour

	for t := -31; t >= -60; t-- {
		logTime := nowTime.Add(time.Duration(t) * oneDay)
		logFilename := writer.dstFile + "." + logTime.Format("2006-01-02")
		if _, e := os.Lstat(logFilename); e == nil {
			os.Remove(logFilename)
			continue
		}
		break
	}
}
