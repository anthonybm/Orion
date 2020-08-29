package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewOrionLogger returns an instance of a Zap logger configured with logFlag level, name of orionRuntime, and outputPath
func NewOrionLogger(logFlag string, orionRuntime string, outputPath string) (*zap.Logger, *os.File, error) {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{outputPath},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalColorLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	filename := orionRuntime + ".json"
	filePath, _ := filepath.Abs(outputPath + "/" + filename)

	// Ensure directory path exists and if not create it
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		os.MkdirAll(outputPath, 0700)
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("[Logging] File does not exist")
		}
		// return nil, err
		log.Fatal(err)
		panic("Log failure!")
	}

	fileEncoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(cfg.EncoderConfig)

	level := zap.InfoLevel
	if logFlag == "debug" {
		fmt.Println("[Logging] Orion output filepath: " + filePath)
		level = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(f), level),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
	)
	logger := zap.New(core)

	zap.ReplaceGlobals(logger)
	defer logger.Sync() // flushes buffer, if any
	zap.L().Debug("Global logger established")

	logger.Debug("Starting logger.")

	return logger, f, nil
}
