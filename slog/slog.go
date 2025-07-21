package slogging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
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
		ctxKeys []string
	}

	SlogOpts struct {
		// Component enriches each log line with a componenent key/value.
		// Useful for aggregating/filtering with your log collector.
		Component string
		// Handler allows overriding of the defult Logfmt handler.
		Handler slog.Handler
		// Minimal level to log. Defaults to Info.
		// No effect when passing a custom handler.
		LogLevel slog.Level
		// Add source location to each log line. Defaults to false.
		// No effect when passing a custom handler.
		IncludeSource bool
		// CtxKeys are the known keys to be used for logging context values.
		CtxKeys []string
	}
)

// NewSlog creates a new Slog logger instance.
func NewSlog(o SlogOpts) *Slog {
	if o.Handler == nil {
		o.Handler = buildDefaultHandler(os.Stderr, o.LogLevel, o.IncludeSource)
	}

	slogger := slog.New(o.Handler)
	if o.Component != "" {
		slogger = slogger.With("component", o.Component)
	}
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

// Global logger instance
var Global *Slog = NewSlog(SlogOpts{
	LogLevel: LevelTrace,
})

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

func (s *Slog) logWithCaller(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !s.slogger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])

	if len(args) > 0 {
		for i := 0; i < len(args)-1; i += 2 {
			key, ok := args[i].(string)
			if !ok {
				continue
			}
			record.AddAttrs(slog.Any(key, args[i+1]))
		}
	}

	_ = s.slogger.Handler().Handle(ctx, record)
}

func (s *Slog) logWithCallerCtx(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !s.slogger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])

	attrs := s.extractContextKeys(ctx, args...)
	record.AddAttrs(attrs...)

	_ = s.slogger.Handler().Handle(ctx, record)
}

func (s *Slog) Tracef(msg string, args ...any) {
	s.logWithCaller(context.Background(), LevelTrace, msg, args...)
}

func (s *Slog) TraceCtxf(ctx context.Context, msg string, args ...any) {
	s.logWithCallerCtx(ctx, LevelTrace, msg, args...)
}

func (s *Slog) Debugf(msg string, args ...any) {
	s.logWithCaller(context.Background(), slog.LevelDebug, msg, args...)
}

func (s *Slog) DebugCtxf(ctx context.Context, msg string, args ...any) {
	s.logWithCallerCtx(ctx, slog.LevelDebug, msg, args...)
}

func (s *Slog) Infof(msg string, args ...any) {
	s.logWithCaller(context.Background(), slog.LevelInfo, msg, args...)
}

func (s *Slog) InfoCtxf(ctx context.Context, msg string, args ...any) {
	s.logWithCallerCtx(ctx, slog.LevelInfo, msg, args...)
}

func (s *Slog) Warnf(msg string, args ...any) {
	s.logWithCaller(context.Background(), slog.LevelWarn, msg, args...)
}

func (s *Slog) WarnCtxf(ctx context.Context, msg string, args ...any) {
	s.logWithCallerCtx(ctx, slog.LevelWarn, msg, args...)
}

func (s *Slog) Errorf(msg string, args ...any) {
	s.logWithCaller(context.Background(), slog.LevelError, msg, args...)
}

func (s *Slog) ErrorCtxf(ctx context.Context, msg string, args ...any) {
	s.logWithCallerCtx(ctx, slog.LevelError, msg, args...)
}

func (s *Slog) extractContextKeys(ctx context.Context, args ...any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(s.ctxKeys)+len(args)/2)

	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}

	for _, key := range s.ctxKeys {
		if val, ok := ctx.Value(key).(string); ok {
			attrs = append(attrs, slog.String(key, val))
		}
	}

	return attrs
}
