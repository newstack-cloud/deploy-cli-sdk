package commands

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupLogger creates a zap logger instance that writes to a file
// named after the CLI (e.g. "celerity.log", "bluelink.log").
func SetupLogger(cliName string) (*zap.Logger, *os.File, error) {
	logFileName := fmt.Sprintf("%s.log", cliName)
	logFileHandle, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	writerSync := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(logFileHandle),
	)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		writerSync,
		zap.DebugLevel,
	)
	logger := zap.New(core)
	return logger, logFileHandle, nil
}
