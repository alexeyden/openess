package log

import (
	"log"
	"os"
)

const (
	LOG_OFF = -1
	LOG_ERROR  = 0
	LOG_INFO  = 1
	LOG_DEBUG = 2
)

type Logger struct {
    inner *log.Logger
    level int
}

var LOGGER *Logger

func Init(level int) {
    inner := log.New(os.Stderr, "", log.LstdFlags)
    LOGGER = new(Logger)
    LOGGER.inner = inner
    LOGGER.level = level
}

func PrError(m string, args ...any) {
    PrLog(LOG_INFO, "\033[31mERROR\033[0m\t", m, args...)
}

func PrInfo(m string, args ...any) {
    PrLog(LOG_INFO, "\033[97mINFO\033[0m\t", m, args...)
}

func PrDebug(m string, args ...any) {
    PrLog(LOG_DEBUG, "\033[35mDEBUG\033[0m\t", m, args...)
}

func PrLog(level int, prefix string, m string, args ...any) {
    if LOGGER.inner == nil {
        return;
    }

    if LOGGER.level < level  {
        return
    }

    LOGGER.inner.SetPrefix(prefix)
    LOGGER.inner.Printf(m, args...)

}
