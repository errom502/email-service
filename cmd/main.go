package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/errom502/email-service/cmd/config"
	"github.com/errom502/email-service/internal/app"
	"github.com/errom502/email-service/internal/tools"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	AppName      = "email-service"
	AppRelease   = "unspecified"
	AppCommit    = "unspecified"
	AppBuildTime = time.Now().Format("06-01-02_15:04:05")
)

func main() {
	// Парсинг конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal("can't parse app config")
	}

	if cfg.MaxCpu > 0 {
		runtime.GOMAXPROCS(cfg.MaxCpu)
	}

	applicationInfo := &app.AppInfo{
		Name:      AppName,
		BuildTime: AppBuildTime,
		Commit:    AppCommit,
		Release:   AppRelease,
	}

	logger, err := initLogger(applicationInfo, cfg)
	if err != nil {
		log.Fatalf("can't init logger: %v", err)
		return
	}

	defer func() {
		if e := recover(); e != nil {
			logger.Error("panic error", zap.Error(fmt.Errorf("%s", e)))
		}
	}()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// Обработка сигналов системы
	signalHandler(logger, cancelCtx)

	application := app.NewApp(cfg, applicationInfo, logger)
	if err = application.Run(ctx); err != nil {
		switch {
		case errors.Is(err, app.ErrAppStartup):
			logger.Error("can't run application", zap.Error(err))
		case errors.Is(err, app.ErrAppShutdownWithError):
			logger.Error("application is shutdown with error", zap.Error(err))
		case errors.Is(err, app.ErrAppShutdownNormal):
			fallthrough
		default:
			logger.Warn("application is shutdown")
		}
	}

}

func initLogger(info *app.AppInfo, cfg *config.Config) (tools.Logger, error) {
	// TODO: в бущем подключить к удаленной системе для просмотра логов
	level := parseLogLevel(cfg.LogLevel)

	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: cfg.DevMode,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       "ts",
			LevelKey:      "level",
			NameKey:       "logger",
			CallerKey:     "caller",
			MessageKey:    "msg",
			StacktraceKey: "stacktrace",
			LineEnding:    zapcore.DefaultLineEnding,
			EncodeLevel:   zapcore.LowercaseLevelEncoder,
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},

		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap config: %w", err)
	}

	logger = logger.With(
		zap.String("service", info.Name),
		zap.String("env", cfg.Environment),
	)

	return logger, err
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.WarnLevel
	}
}

// signalHandler обработчик сигналов системы
func signalHandler(logger tools.Logger, cancelFunc context.CancelFunc) {
	osSigCh := make(chan os.Signal, 1)

	signal.Notify(
		osSigCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		defer signal.Stop(osSigCh)
		s := <-osSigCh
		switch s {
		case syscall.SIGHUP:
			logger.Info("Received signal SIGHUP! Application shutdown")
		case syscall.SIGINT:
			logger.Info("Received signal SIGINT! Application shutdown")
		case syscall.SIGQUIT:
			logger.Info("Received signal SIGQUIT! Application shutdown")
		case syscall.SIGTERM:
			logger.Info("Received signal SIGTERM! Application shutdown")
		}
		cancelFunc()
	}()
}
