package log

import (
	"errors"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	STDOUT     bool   // stdout
	File       string // log out put file path, empty means no log file
	Level      int8   // debug -1 | info 0 (default) | warn 1 | error 2
	MaxAge     int    // 保存的天数, 默认不删除
	MaxSize    int    // 单个文件大小 MB
	MaxBackups int    // 最多保留的备份数
	Compress   bool   // 是否压缩
	JsonFormat bool   // 是否用 json 格式
}

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

func GetID(id uint64) *zap.Field {
	return &zap.Field{
		Key:       "ID",
		Type:      zapcore.Uint64Type,
		Integer:   0,
		String:    "",
		Interface: id,
	}
}

func Init(config Config) error {

	var wss []zapcore.WriteSyncer
	if len(config.File) > 0 {
		hook := lumberjack.Logger{
			Filename:   config.File,    // 日志文件路径
			MaxSize:    config.MaxSize, // megabytes
			MaxAge:     config.MaxAge,
			MaxBackups: config.MaxBackups, // 最多保留300个备份
			LocalTime:  false,
			Compress:   config.Compress, // 是否压缩 disabled by default
		}
		wss = append(wss, zapcore.AddSync(&hook))
	}

	if config.STDOUT {
		wss = append(wss, zapcore.AddSync(os.Stdout))
	}

	if len(wss) == 0 {
		return errors.New("write syncer needed")
	}

	cfg := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	var enc zapcore.Encoder
	if config.JsonFormat {
		enc = zapcore.NewJSONEncoder(cfg)
	} else {
		enc = zapcore.NewConsoleEncoder(cfg)
	}

	switch zapcore.Level(config.Level) {
	case zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel:
	default:
		config.Level = int8(zapcore.InfoLevel)
	}

	Logger = zap.New(zapcore.NewCore(enc, zapcore.NewMultiWriteSyncer(wss...), zapcore.Level(config.Level)), zap.AddCaller())
	Sugar = Logger.Sugar()

	return nil
}

func InitDevelop(config Config) {
	l, err := zap.NewDevelopment(nil)
	if err != nil {
		panic(err)
	}

	Sugar = l.Sugar()
}
