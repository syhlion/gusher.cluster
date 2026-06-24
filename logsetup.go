package main

import (
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// LogSetup 把「輸出目的地（stdout/file/both）＋ 格式 ＋ 輪替」這件事集中設定,
// 並產出兩個寫到「同一個目的地」的 logger:
//   - Logrus:gusher 自身的 app 日誌(沿用既有 WithFields/WithError 呼叫點)
//   - Slog  :傳給 redisocket 引擎(引擎已改吃 *slog.Logger)
//
// 取代原本壞掉的 GUSHER_LOG_FORMATTER(從未被賦值),改由下列 env 驅動。
type LogSetup struct {
	Logrus *Logger
	Slog   *slog.Logger
	Close  func() error
}

// setupLoggingFromEnv 依環境變數建立 LogSetup:
//
//	GUSHER_LOG_OUTPUT   stdout(預設) | file | both
//	GUSHER_LOG_FILE     file/both 時的路徑(預設 ./gusher.log)
//	GUSHER_LOG_FORMAT   json(預設) | text
//	GUSHER_LOG_MAX_SIZE_MB / _MAX_BACKUPS / _MAX_AGE_DAYS / _COMPRESS  輪替設定
func setupLoggingFromEnv() (*LogSetup, error) {
	output := strings.ToLower(getenvDefault("GUSHER_LOG_OUTPUT", "stdout"))
	file := getenvDefault("GUSHER_LOG_FILE", "./gusher.log")
	format := strings.ToLower(getenvDefault("GUSHER_LOG_FORMAT", "json"))

	var (
		w      io.Writer = os.Stdout
		closer           = func() error { return nil }
	)
	if output == "file" || output == "both" {
		lj := &lumberjack.Logger{
			Filename:   file,
			MaxSize:    atoiDefault("GUSHER_LOG_MAX_SIZE_MB", 100),
			MaxBackups: atoiDefault("GUSHER_LOG_MAX_BACKUPS", 7),
			MaxAge:     atoiDefault("GUSHER_LOG_MAX_AGE_DAYS", 30),
			Compress:   strings.ToLower(os.Getenv("GUSHER_LOG_COMPRESS")) == "true",
		}
		if output == "both" {
			w = io.MultiWriter(os.Stdout, lj) // stdout 與檔案同時寫
		} else {
			w = lj
		}
		closer = lj.Close
	}

	// gusher app 日誌（logrus）導到同一個 writer
	ll := logrus.New()
	ll.SetOutput(w)
	if format == "text" {
		ll.SetFormatter(&logrus.TextFormatter{})
	} else {
		ll.SetFormatter(&logrus.JSONFormatter{})
	}

	// 引擎日誌（slog）寫到同一個 writer
	var sh slog.Handler
	if format == "text" {
		sh = slog.NewTextHandler(w, nil)
	} else {
		sh = slog.NewJSONHandler(w, nil)
	}

	return &LogSetup{
		Logrus: &Logger{ll},
		Slog:   slog.New(sh),
		Close:  closer,
	}, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoiDefault(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}
