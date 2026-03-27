package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once          sync.Once
	zapLogger     *zap.Logger
	logLevel      zapcore.Level
	callerEncoder zapcore.CallerEncoder
	consoleFields []string
)

func LogDebug(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

func LogInfo(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func LogFlowLog(format string, args ...interface{}) {
	log.Printf("[FLOW_LOG] "+format, args...)
}

func LogError(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

func InitLogger() {
	once.Do(func() {
		log.SetFlags(log.Ldate | log.Ltime)
		initZapLogger()
		Info("INIT_LOGGER")
	})
}

func initZapLogger() {
	log.Printf("log level: %v", logLevel.CapitalString())
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "name",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		FunctionKey:    "function",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   getCallerEncoder(),
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(getSyncers()...),
		logLevel,
	)
	zapLogger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).With(
		zap.String("hostname", *getHost()),
	)
}

func getHost() *string {
	ret, err := os.Hostname()
	if err != nil {
		log.Print(err)
		return nil
	}
	return &ret
}

func getCallerEncoder() zapcore.CallerEncoder {
	if callerEncoder != nil {
		return callerEncoder
	}
	return zapcore.ShortCallerEncoder
}

func getSyncers() []zapcore.WriteSyncer {
	var syncers []zapcore.WriteSyncer
	return append(syncers, zapcore.AddSync(os.Stdout))
}

func GetLogger() *zap.Logger {
	return zapLogger
}

func LogStruct(s string, v interface{}) {
	if v == nil {
		fmt.Printf("----LogStruct-----%s---------\n", s)
		fmt.Println("v is nil")
		return
	}

	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			fmt.Printf("----LogStruct-----%s---------\n", s)
			fmt.Println("v is a nil pointer")
			return
		}
		val = val.Elem()
	}

	typ := val.Type()
	indent := "  "
	fmt.Printf("----LogStruct-----%s---------\n", s)

	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldName := typ.Field(i).Name

			if !field.CanInterface() {
				continue
			}

			fmt.Printf("%s%s: %v\n", indent, fieldName, field.Interface())
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			value := val.MapIndex(key)
			fmt.Printf("%s%s: %v\n", indent, key, value.Interface())
		}
	default:
		fmt.Printf("%s%v\n", indent, val.Interface())
	}
}

// テスト用
func StructToJson(s string, v interface{}) {
	jsonString, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("----StructToJson-----%s---------\n", s)
	fmt.Println(string(jsonString))
}

func FormatJSONPayload(s string, payload []byte) {
	fmt.Println("FormatJSONPayload1")

	cleanPayload := string(payload)
	fmt.Println("Cleaned payload:", cleanPayload)

	var prettyJSON map[string]interface{}
	err := json.Unmarshal([]byte(cleanPayload), &prettyJSON)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
		return
	}
	fmt.Println("FormatJSONPayload3")

	pretty, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		fmt.Println("MarshalIndent error:", err)
		return
	}
	fmt.Printf("----FormatJSONPayload-----%s---------\n", s)
	fmt.Println(string(pretty))
}
