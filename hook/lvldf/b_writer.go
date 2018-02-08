package lvldf

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

import (
	"golang.org/x/sys/unix"
)

import (
	"github.com/sirupsen/logrus"
)

const (
	bufSizeForWriter = 1024 * 10
)

type mutexWriter struct {
	sync.Mutex

	level logrus.Level
	hook  *levelDividedFileLogger

	logFd       *os.File
	logFileName string

	bufMutex  *sync.Mutex
	bufWriter *bufio.Writer

	lockFd       int
	lockFileName string

	lastRotateTime time.Time
}

func newMutexWriter(level logrus.Level, logger *levelDividedFileLogger) *mutexWriter {
	w := &mutexWriter{
		level:    level,
		hook:     logger,
		bufMutex: &sync.Mutex{},
	}
	fileName := fmt.Sprintf("%s.%s", logger.logName, levelFileSuffixMap[level])
	w.logFileName = filepath.Join(logger.logDirectory, fileName)

	if w.hook.bufferEnable {
		go w.syncPerSeconds()
	}

	return w
}

func (w *mutexWriter) writeMessage(level logrus.Level, logTime time.Time, message string) error {

	if w.hook.rotateEnable {
		w.checkAndDoRotate(logTime)
	}

	if w.hook.bufferEnable && w.bufWriter != nil {
		w.bufMutex.Lock()
		_, err := w.bufWriter.WriteString(message)
		w.bufMutex.Unlock()
		return err
	} else if w.logFd != nil {
		_, err := w.logFd.WriteString(message)
		return err
	}

	return nil
}

func (w *mutexWriter) syncPerSeconds() {
	t := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-t.C:
			w.syncOnce()
		}
	}
}

func (w *mutexWriter) syncOnce() {

	if w.bufWriter != nil {
		w.bufMutex.Lock()
		w.bufWriter.Flush()
		w.bufMutex.Unlock()
	}
}

func (w *mutexWriter) startLogWriter() error {

	if w.lockFd == 0 {
		lockSign := fmt.Sprintf("%x", md5.Sum([]byte(w.logFileName)))
		parts := strings.Split(w.logFileName, "/")
		w.lockFileName = filepath.Join(os.TempDir(), parts[len(parts)-1]+"."+lockSign+".lock")
		w.lockFd, _ = unix.Open(w.lockFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	}

	if fd, err := w.createLogFile(); err != nil {
		return err
	} else {
		w.setFd(fd)
		return nil
	}
}

func (w *mutexWriter) createLogFile() (*os.File, error) {
	return os.OpenFile(w.logFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0665)
}

func (w *mutexWriter) setFd(fd *os.File) {
	if fd == nil {
		return
	}

	if w.logFd != nil {
		w.flush()
		w.destroy()
	}

	w.logFd = fd

	if w.hook.bufferEnable {
		if w.bufWriter == nil {
			w.bufWriter = bufio.NewWriterSize(w.logFd, bufSizeForWriter)
		} else {
			w.bufWriter.Reset(w.logFd)
		}
	}

	if fdInfo, _ := w.logFd.Stat(); fdInfo != nil {
		w.lastRotateTime = fdInfo.ModTime()
	} else {
		w.lastRotateTime = time.Now()
	}
}

func (w *mutexWriter) checkShouldRotate(logTime time.Time) bool {
	if !w.hook.rotateEnable {
		return false
	}

	switch w.hook.rotateBy {
	case rotateByDay:
		/*

			if w.lastRotateTime.Minute() != logTime.Minute() {
				return true
			}
		*/
		if w.lastRotateTime.Day() != logTime.Day() {
			return true
		}
	}

	if w.lastRotateTime.Month() != logTime.Month() {
		return true
	}
	if w.lastRotateTime.Year() != logTime.Year() {
		return true
	}
	return false

}

func (w *mutexWriter) checkAndDoRotate(logTime time.Time) error {
	w.Lock()
	defer w.Unlock()

	if w.logFd == nil {
		w.startLogWriter()
		return nil
	}

	if !w.checkShouldRotate(logTime) {
		return nil
	}

	w.flush()
	w.destroy()

	var err error

	doBackup := func() error {
		_, e1 := os.Lstat(w.logFileName)
		backupFile := w.logFileName + "." + w.lastRotateTime.Format("2006-01-02")

		_, e2 := os.Lstat(backupFile)

		if e1 == nil && e2 != nil {
			if err = os.Rename(w.logFileName, backupFile); err != nil {
				log.Printf("[ERROR] split log error, rename fail, %s", err.Error())
			}
			return err
		}

		if e1 == nil && e2 == nil {
			log.Printf("[WARN] %v may be has backuped to %v", w.logFileName, backupFile)
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

	if e := flockUtil.Lock(uintptr(w.lockFd)); e == nil {
		err = doBackup()
		flockUtil.UnLock(uintptr(w.lockFd))
	} else {
		log.Printf("[ERROR] do log rotate fail, flock error=%v", e)
	}

	if err != nil {
		log.Printf("[ERROR] daily rotate log %v error=%v", w.logFileName, err)
	}

	err = w.startLogWriter()

	go w.deleteOldLog()
	return err
}

func (w *mutexWriter) destroy() {
	if w.logFd != nil {
		w.logFd.Close()
		w.logFd = nil
	}
}

func (w *mutexWriter) flush() {
	if w.logFd != nil {
		if w.hook.bufferEnable {
			if w.bufWriter != nil {
				w.bufMutex.Lock()
				w.bufWriter.Flush()
				w.bufMutex.Unlock()
			}
		} else {
			w.logFd.Sync()
		}
	}
}

func (w *mutexWriter) deleteOldLog() {

	nowTime := time.Now()
	oneDay := 24 * time.Hour

	for t := -31; t >= -60; t-- {
		logTime := nowTime.Add(time.Duration(t) * oneDay)
		logFilename := w.logFileName + "." + logTime.Format("2006-01-02")
		if _, e := os.Lstat(logFilename); e == nil {
			os.Remove(logFilename)
			continue
		}
		break
	}
}
