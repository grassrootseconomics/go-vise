package slogging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// Logger defines the interface for structured logging
type Logger interface {
	Tracef(msg string, args ...any)
	TraceCtxf(ctx context.Context, msg string, args ...any)
	Debugf(msg string, args ...any)
	DebugCtxf(ctx context.Context, msg string, args ...any)
	Infof(msg string, args ...any)
	InfoCtxf(ctx context.Context, msg string, args ...any)
	Warnf(msg string, args ...any)
	WarnCtxf(ctx context.Context, msg string, args ...any)
	Errorf(msg string, args ...any)
	ErrorCtxf(ctx context.Context, msg string, args ...any)
}

const (
	LevelTrace slog.Level = slog.Level(-8)
)

var _ Logger = (*Slog)(nil)

type (
	Slog struct {
		slogger *slog.Logger
		ctxKeys []any
	}

	SlogOpts struct {
		// Handler allows overriding of the defult Logfmt handler.
		Handler slog.Handler
		// Minimal level to log. Defaults to Info.
		// No effect when passing a custom handler.
		LogLevel slog.Level
		// Add source location to each log line. Defaults to false.
		// No effect when passing a custom handler.
		IncludeSource bool
		// CtxKeys are the known keys to be used for logging context values.
		CtxKeys []any
	}
)

var (
	defaultLogger *Slog
	once          sync.Once
)

// NewSlog creates a new Slog logger instance.
func NewSlog(o SlogOpts) *Slog {
	if o.Handler == nil {
		o.Handler = buildDefaultHandler(os.Stderr, o.LogLevel, o.IncludeSource)
	}

	slogger := slog.New(o.Handler)
	return &Slog{
		slogger: slogger,
		ctxKeys: o.CtxKeys,
	}
}

// With creates a new logger with additional attributes
func (s *Slog) With(args ...any) *Slog {
	return &Slog{
		slogger: s.slogger.With(args...),
		ctxKeys: s.ctxKeys,
	}
}

func SetGlobal(logger *Slog) {
	defaultLogger = logger
}

func Get() *Slog {
	once.Do(func() {
		if defaultLogger == nil {
			defaultLogger = NewSlog(SlogOpts{
				LogLevel: LevelTrace,
			})
		}
	})
	return defaultLogger
}

func buildDefaultHandler(w io.Writer, level slog.Level, includeSource bool) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: includeSource,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				switch a.Value.Any().(slog.Level) {
				// stdlib slog does not support TRACE level, so we map it to a custom string.
				case LevelTrace:
					return slog.String(slog.LevelKey, "TRACE")
				}
			}
			return a
		},
	})
}

func (s *Slog) Tracef(msg string, args ...any) {
	attrs := s.argsToAttrs(args)
	s.slogger.LogAttrs(context.Background(), LevelTrace, msg, attrs...)
}

func (s *Slog) TraceCtxf(ctx context.Context, msg string, args ...any) {
	attrs := s.extractContextAndArgs(ctx, args...)
	s.slogger.LogAttrs(ctx, LevelTrace, msg, attrs...)
}

func (s *Slog) Debugf(msg string, args ...any) {
	attrs := s.argsToAttrs(args)
	s.slogger.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}

func (s *Slog) DebugCtxf(ctx context.Context, msg string, args ...any) {
	attrs := s.extractContextAndArgs(ctx, args...)
	s.slogger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

func (s *Slog) Infof(msg string, args ...any) {
	attrs := s.argsToAttrs(args)
	s.slogger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

func (s *Slog) InfoCtxf(ctx context.Context, msg string, args ...any) {
	attrs := s.extractContextAndArgs(ctx, args...)
	s.slogger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

func (s *Slog) Warnf(msg string, args ...any) {
	attrs := s.argsToAttrs(args)
	s.slogger.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

func (s *Slog) WarnCtxf(ctx context.Context, msg string, args ...any) {
	attrs := s.extractContextAndArgs(ctx, args...)
	s.slogger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

func (s *Slog) Errorf(msg string, args ...any) {
	attrs := s.argsToAttrs(args)
	s.slogger.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

func (s *Slog) ErrorCtxf(ctx context.Context, msg string, args ...any) {
	attrs := s.extractContextAndArgs(ctx, args...)
	s.slogger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

func (s *Slog) extractContextAndArgs(ctx context.Context, args ...any) []slog.Attr {
	attrs := s.argsToAttrs(args)

	for _, key := range s.ctxKeys {
		if val := ctx.Value(key); val != nil {
			keyStr := fmt.Sprintf("%v", key)
			attrs = append(attrs, slog.Any(keyStr, val))
		}
	}

	return attrs
}

func (s *Slog) argsToAttrs(args []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return attrs
}
