package lib

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var Logger = &FileLogger{}

func NewLogger(path, name string, level int) *FileLogger {
	ins := &FileLogger{}
	ins.Init(path, name, level)

	return ins
}

// Log levels
const (
	LevelNone   = 0x00
	LevelError  = 0x01
	LevelWarn   = 0x02
	LevelAccess = 0x04
	LevelInfo   = 0x08
	LevelDebug  = 0x10
	LevelAll    = 0xFF
)

type FileLogger struct {
	loggerMap map[string]*log.Logger
	curDate   map[string]string
	rootPath  string
	logName   string
	logLevel  int
	lock      sync.RWMutex
}

func (this *FileLogger) Init(rootPath, logName string, logLevel int) {
	this.rootPath = rootPath
	this.logName = logName
	this.curDate = make(map[string]string)
	this.loggerMap = make(map[string]*log.Logger)
	this.logLevel = logLevel

	os.MkdirAll(this.rootPath, 0777)
}

func (this *FileLogger) getLogger(logName string) (*log.Logger, error) {
	nowDate := time.Now().Format("20060102")
	filePath := this.rootPath + "/" + logName + "." + nowDate
	this.lock.RLock()
	retLogger, ok := this.loggerMap[logName]
	curDate, ok := this.curDate[logName]
	if !ok || nowDate != curDate {
		this.lock.RUnlock()
		this.lock.Lock()
		defer this.lock.Unlock()

		retLoggerRetry, ok := this.loggerMap[logName]
		curDateRetry, ok := this.curDate[logName]
		//双重判断，减少抢锁
		if !ok || nowDate != curDateRetry {
			fd, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
			if err != nil {
				return nil, err
			}
			//创建文件的时候指定777权限不管用，所有只能在显式chmod
			fd.Chmod(0777)
			this.loggerMap[logName] = log.New(fd, "", 0)
			this.curDate[logName] = nowDate
			fmt.Println("new logger:", filePath)

			retLogger = this.loggerMap[logName]
		} else {
			retLogger = retLoggerRetry
		}
	} else {
		this.lock.RUnlock()
	}

	return retLogger, nil
}

func (this *FileLogger) writeLog(logName string, v ...interface{}) {
	go this._writeLog(logName, v...)
}

func (this *FileLogger) _writeLog(logName string, v ...interface{}) { // {{{
	logger, err := this.getLogger(logName)
	if err != nil {
		fmt.Println("log failed", err)
		return
	}
	msgstr := ""
	for _, msg := range v {
		if msg1, ok := msg.(map[string]interface{}); ok {
			//map每次输出的顺序是随机的，以下保证每次输出的顺序一致，如果map比较大，可能有一定性能损耗
			var keys []string
			for k := range msg1 {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				msgstr = msgstr + fmt.Sprintf("%s[%+v] ", k, msg1[k])
			}
		} else {
			msgstr = msgstr + fmt.Sprintf("%+v ", msg)
		}
	}
	msgstr = strings.TrimRight(msgstr, ",")
	timeNow := time.Now().Format("2006-01-02 15:04:05") //go的坑，必须是2006-01-02 15:04:05
	logger.Printf("time[%s] %s\n", timeNow, msgstr)
} // }}}

func (this *FileLogger) Debug(v ...interface{}) { // {{{
	if this.logLevel&LevelDebug == 0 {
		return
	}

	fmt.Printf("debug:")
	for _, val := range v {
		fmt.Printf(" %#v ", val)
	}
	fmt.Println("")
	this.writeLog(this.logName+".debug", v...)
} // }}}

func (this *FileLogger) Info(v ...interface{}) { // {{{
	if this.logLevel&LevelInfo == 0 {
		return
	}
	this.writeLog(this.logName+".info", v...)
} // }}}

func (this *FileLogger) Access(v ...interface{}) { // {{{
	if this.logLevel&LevelAccess == 0 {
		return
	}
	this.writeLog(this.logName, v...)
} // }}}

func (this *FileLogger) Warn(v ...interface{}) { // {{{
	if this.logLevel&LevelWarn == 0 {
		return
	}
	this.writeLog(this.logName+".warn", v...)
} // }}}

func (this *FileLogger) Error(v ...interface{}) { // {{{
	if this.logLevel&LevelError == 0 {
		return
	}
	this.writeLog(this.logName+".error", v...)
} // }}}

func (this *FileLogger) Other(logname string, v ...interface{}) {
	this.writeLog(logname, v...)
}
