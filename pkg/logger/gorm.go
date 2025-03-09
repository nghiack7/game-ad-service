package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

type ZapGormLogger struct {
	zapLogger *zap.Logger
	logLevel  logger.LogLevel
}

func NewZapGormLogger(zapLogger *zap.Logger, logLevel logger.LogLevel) *ZapGormLogger {
	return &ZapGormLogger{
		zapLogger: zapLogger,
		logLevel:  logLevel,
	}
}

func (l *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		l.zapLogger.Sugar().Infof(msg, data...)
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.zapLogger.Sugar().Warnf(msg, data...)
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		l.zapLogger.Sugar().Errorf(msg, data...)
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= 0 {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	if err != nil {
		l.zapLogger.Sugar().Errorf("SQL: %s, Rows affected: %d, Error: %s, Elapsed: %s", sql, rows, err, elapsed)
	} else if elapsed > time.Second {
		l.zapLogger.Sugar().Warnf("Slow query detected. SQL: %s, Rows affected: %d, Elapsed: %s", sql, rows, elapsed)
	} else {
		l.zapLogger.Sugar().Debugf("SQL: %s, Rows affected: %d, Elapsed: %s", sql, rows, elapsed)
	}
}
