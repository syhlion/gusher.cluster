package main

import (
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// LogSetup 把「輸出目的地（stdout/file/both）＋ 格式 ＋ 輪替」集中設定。
// 全程 slog:App 給 gusher 自身日誌(slog-backed 的 *Logger)、Slog 給 redisocket
// 引擎,兩者同一個 handler/writer。取代原本壞掉的 GUSHER_LOG_FORMATTER。
type LogSetup struct {
	App   *Logger
	Slog  *slog.Logger
	Close func() error
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

	opts := &slog.HandlerOptions{Level: logLevel}
	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	sl := slog.New(h)

	return &LogSetup{App: newLogger(sl), Slog: sl, Close: closer}, nil
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
