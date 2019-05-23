package logrus_extension

/**

logrus hook，按照日志级别分别记录到对应的文件种
-------
1、按日志级别输出；
2、支持按天切分日志；
3、自动清理过期日志（30天）；

*/

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/juju/fslock"
	"github.com/sirupsen/logrus"
	"github.com/xgo11/stdlog"
)

const (
	lvlMinLevel = logrus.ErrorLevel
)

type rolloverAble interface {
	ShouldRollover(entry *logrus.Entry) bool
	DoRollover(entry *logrus.Entry) error
}

type entryWriter interface {
	WriteEntry(entry *logrus.Entry, entryBytes []byte) error
}

type lvlMutexWriter struct {
	outputFd   *os.File // log output file fd
	outputFile string   // log output file name
	lockFile   string   // file lock file name
	wLock      *sync.Mutex
	interval   int64
	format     string
	when       string
	modTime    int64 // output fd modify time
	rolloverAt int64 // next rotate time
}

var (
	log               = stdlog.Std
	rotateIntervalMap = map[string]int64{ // rotate interval
		"D": 60 * 60 * 24, // one day
		"H": 60 * 60,      // one hour
		"M": 60,           // one minute
		"S": 1,            // one second
	}
	rotateSuffixFormat = map[string]string{ // rotate backup file suffix format
		"D": "2006-01-02",        // one day
		"H": "2006-01-02_15",     // one hour
		"M": "2006-01-02_1504",   // one minute
		"S": "2006-01-02_150405", // one second
	}
)

func newRotateWriter(level logrus.Level, dir, file string, rotateInterval string) entryWriter {
	suffix := levelNameDict[level]
	if suffix == "" {
		suffix = level.String()
	}
	outputFile := filepath.Join(dir, file+"."+suffix)
	lockSign := fmt.Sprintf("%x", md5.Sum([]byte(outputFile)))
	parts := strings.Split(outputFile, "/")
	lockFile := filepath.Join(os.TempDir(), parts[len(parts)-1]+"."+lockSign+".lock")

	w := &lvlMutexWriter{outputFile: outputFile, lockFile: lockFile}

	if rotateInterval == "" {
		log.Panicf("invalid logger conf, rotateInterval is empty")
	}
	iLst := rotateInterval[len(rotateInterval)-1:]
	if unit, ok := rotateIntervalMap[iLst]; !ok {
		log.Panicf("invalid logger conf, rotateInterval:%v", rotateInterval)
	} else {
		w.format = rotateSuffixFormat[iLst]
		if iLst == rotateInterval {
			w.interval = unit
		} else {
			if i, e := strconv.ParseInt(strings.TrimRight(rotateInterval, iLst), 10, 64); e != nil {
				log.Panicf("invalid logger conf, rotateInterval:%v", rotateInterval)
			} else {
				w.interval = unit * i
			}
		}
	}
	w.when = iLst
	w.wLock = &sync.Mutex{}

	if stat, err := os.Lstat(w.outputFile); err == nil {
		w.rolloverAt = w.computeRolloverAt(stat.ModTime().In(time.Local).Unix())
	} else {
		w.rolloverAt = w.computeRolloverAt(time.Now().In(time.Local).Unix())
	}

	var _ rolloverAble = w
	var _ entryWriter = w
	//debugF("lockFile=%s", w.lockFile)

	return w

}

func (trw *lvlMutexWriter) ShouldRollover(entry *logrus.Entry) bool {
	return entry.Time.Unix() >= trw.rolloverAt
}

func (trw *lvlMutexWriter) DoRollover(entry *logrus.Entry) (err error) {
	debugF("DoRollover ...")
	var fs = fslock.New(trw.lockFile)

	if err = fs.Lock(); err == nil {
		defer func() {
			_ = fs.Unlock()
		}()

		if trw.outputFd != nil { // close fd
			_ = trw.outputFd.Sync()
			_ = trw.outputFd.Close()
			trw.outputFd = nil
		}

		if oInfo, e1 := os.Lstat(trw.outputFile); e1 == nil { // file exists
			if modTime := oInfo.ModTime().In(time.Local).Unix(); modTime <= trw.rolloverAt { // file should rollover
				backTime := trw.rolloverAt
				for backTime >= modTime {
					backTime -= trw.interval
				}
				backupFile := trw.outputFile + "." + time.Unix(backTime, 0).In(time.Local).Format(trw.format)

				if _, e2 := os.Lstat(backupFile); e2 != nil {
					defer trw.deleteOldLog()
					err = os.Rename(trw.outputFile, backupFile)
				}

			}
		}
	}

	if now := entry.Time.Unix(); err == nil && trw.rolloverAt <= now {
		trw.rolloverAt = trw.computeRolloverAt(now)
	}

	return
}

func (trw *lvlMutexWriter) WriteEntry(entry *logrus.Entry, entryBytes []byte) (err error) {
	trw.wLock.Lock()
	defer trw.wLock.Unlock()
	if trw.ShouldRollover(entry) {
		if err := trw.DoRollover(entry); err != nil {
			log.Warnf("DoRollover[%v] failed, %v", trw.outputFile, err)
		}
	}
	if len(entryBytes) < 1 {
		if entryBytes, err = entry.Logger.Formatter.Format(entry); err != nil {
			return
		}
	}
	if trw.outputFd == nil {
		trw.open()
	}
	_, err = trw.outputFd.Write(entryBytes)
	return
}

func (trw *lvlMutexWriter) open() {
	if trw.outputFd == nil {
		if fd, err := os.OpenFile(trw.outputFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0665); err == nil && fd != nil {
			trw.outputFd = fd
		}
	}
}

func (trw *lvlMutexWriter) computeRolloverAt(currentTime int64) int64 {
	ct := time.Unix(currentTime, 0)
	switch trw.when {
	case "D":
		ct = ct.Truncate(time.Hour)
		if hour := ct.Hour(); hour > 0 {
			ct = ct.Add(time.Duration(-hour) * time.Hour)
		}
	case "H":
		ct = ct.Truncate(time.Hour)
	case "M":
		ct = ct.Truncate(time.Hour)
	case "S":
		ct = ct.Truncate(time.Minute)
	}
	rolloverAt := ct.Unix()
	for rolloverAt <= currentTime {
		rolloverAt += trw.interval
	}

	debugF("computeRolloverAt,[%s] %v", trw.outputFile, time.Unix(rolloverAt, 0))
	return rolloverAt
}

func (trw *lvlMutexWriter) deleteOldLog() {

	baseTime := time.Now().In(time.Local).Add(-30 * 24 * time.Hour)
	prefix := trw.outputFile + "."

	if matches, err := filepath.Glob(prefix + "*"); err == nil {
		for _, path := range matches {
			if suffix := strings.TrimLeft(path, prefix); len(suffix) >= 10 {
				if t, e := time.ParseInLocation("2006-01-02", suffix[0:10], time.Local); e == nil {
					if t.Before(baseTime) {
						_ = os.Remove(path)
					}
				}
			}
		}
	}
}
