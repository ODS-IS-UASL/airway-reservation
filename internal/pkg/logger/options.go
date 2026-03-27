package logger

import (
	"fmt"
	"log"
	"strings"

	"uasl-reservation/internal/pkg/config"

	"go.uber.org/zap/zapcore"
)

type ConsoleType int

const (
	ConsoleTypeAll ConsoleType = iota
	ConsoleTypeError
	ConsoleTypeNone
)

func SetLogLevel(option zapcore.Level) {
	logLevel = option
}

// SetLogLevelByString is set log level. levelStr can use (DEBUG,INFO,WARN,ERROR,FATAL).
func SetLogLevelByString(levelStr string) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(levelStr))
	if err != nil {
		log.Fatalf("%s is invalid level. can use (DEBUG,INFO,WARN,ERROR,FATAL)", levelStr)
	}
	SetLogLevel(level)
}
func SetLogLevelByConfig() {
	config.Load()
	conf := config.GetConfig()
	level := zapcore.DebugLevel
	switch {
	case conf.IsLocal():
		level = zapcore.DebugLevel
	case conf.IsDev():
		level = zapcore.DebugLevel
	case conf.IsStage():
		level = zapcore.InfoLevel
	case conf.IsProd():
		level = zapcore.InfoLevel
	}
	SetLogLevel(level)
}

// SetRepositoryCallerEncoder
// build and set CallerEncoder that build a link to the Repository of the caller's source code.
func SetRepositoryCallerEncoder(urlFormat, revisionOrTag, srcRootDir string) {
	if revisionOrTag == "" || srcRootDir == "" {
		return
	}
	url := fmt.Sprintf(urlFormat, revisionOrTag)
	callerEncoder = buildRepositoryCallerEncoder(srcRootDir, url)
}

func buildRepositoryCallerEncoder(dir, url string) zapcore.CallerEncoder {
	return func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(
			fmt.Sprintf("%v#L%v", strings.Replace(caller.File, dir, url, 1), caller.Line),
		)
	}
}

// SetConsoleField Set the fields to be displayed in the console.
func SetConsoleField(fieldKey ...string) {
	consoleFields = append(consoleFields, fieldKey...)
}
