# logrus-extension
---
一个基于 [logrus](http://github.com/sirupsen/logrus)的Golang日志扩展工具.

---

## 模块介绍

### 1. **hook/lvldf**
> - 按日志级别分别输出到不同的日志文件
> - 自动按天切割日志
> - 多进程记录同一份日志的时候保证切割安全
> - 自动清理过期日志(30天)
> - 支持Yaml配置初始化


配置参考[test.yaml](example/test.yaml)
```
directory: "./logs"
filename: "test"
level: "info"
daily: true
```

效果预览<br>
![效果预览](example/demo.jpg)



### 2. **format**  日志格式化规范
> - Tab分隔的key=value格式
> - 统一 log_time、Host、error、message


---

## 使用示例
参考[main.go](example/main.go)
```
package main

import (
	"errors"
	"time"

	"github.com/plusplus1/logrus-extension"
	"github.com/sirupsen/logrus"
)

func main() {

	logrus_extension.InitFileHook("test.yaml")

	logrus_extension.InitFileHookWithLogger("test2.yaml", "test2")


	// 打印日志到test2
	go func() {

		loggerTest2 := logrus_extension.GetLogger("test2")
		for {
			fields := logrus.Fields{
				"a":     1,
				"b":     "b",
				"e":     errors.New("test error"),
				"c":     "ccc\nccc\tddd",
				"d":     "\tabcd",
				"zh_CN": "中文字符",
			}

			loggerTest2.WithFields(fields).Info("message")
			loggerTest2.WithFields(fields).Debug("message")
			loggerTest2.WithFields(fields).Warn("message")
			loggerTest2.WithFields(fields).Error("message")

			time.Sleep(100 * time.Millisecond)
		}

	}()


	// 打印日志到test
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


```