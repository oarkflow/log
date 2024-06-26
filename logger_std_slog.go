//go:build go1.21
// +build go1.21

package log

import (
	"context"
	"log/slog"
)

func slogAttrEval(e *Entry, a slog.Attr) *Entry {
	if a.Equal(slog.Attr{}) {
		return e
	}
	value := a.Value.Resolve()
	switch value.Kind() {
	case slog.KindGroup:
		if len(value.Group()) == 0 {
			return e
		}
		if a.Key == "" {
			for _, attr := range value.Group() {
				e = slogAttrEval(e, attr)
			}
			return e
		}
		e.buf = append(e.buf, ',', '"')
		e.buf = append(e.buf, a.Key...)
		e.buf = append(e.buf, '"', ':')
		i := len(e.buf)
		for _, attr := range value.Group() {
			e = slogAttrEval(e, attr)
		}
		e.buf[i] = '{'
		e.buf = append(e.buf, '}')
		return e
	case slog.KindBool:
		return e.Bool(a.Key, value.Bool())
	case slog.KindDuration:
		return e.Dur(a.Key, value.Duration())
	case slog.KindFloat64:
		return e.Float64(a.Key, value.Float64())
	case slog.KindInt64:
		return e.Int64(a.Key, value.Int64())
	case slog.KindString:
		return e.Str(a.Key, value.String())
	case slog.KindTime:
		return e.Time(a.Key, value.Time())
	case slog.KindUint64:
		return e.Uint64(a.Key, value.Uint64())
	case slog.KindAny:
		fallthrough
	default:
		return e.Any(a.Key, value.Any())
	}
}

type slogHandler struct {
	logger   Logger
	caller   int
	grouping bool
	groups   int
	entry    Entry
}

func (h slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return &h
	}
	i := len(h.entry.buf)
	for _, attr := range attrs {
		h.entry = *slogAttrEval(&h.entry, attr)
	}
	if h.grouping {
		h.entry.buf[i] = '{'
	}
	h.grouping = false
	return &h
}

func (h slogHandler) WithGroup(name string) slog.Handler {
	if name != "" {
		if h.grouping {
			h.entry.buf = append(h.entry.buf, '{')
		} else {
			h.entry.buf = append(h.entry.buf, ',')
		}
		h.entry.buf = append(h.entry.buf, '"')
		h.entry.buf = append(h.entry.buf, name...)
		h.entry.buf = append(h.entry.buf, '"', ':')
		h.grouping = true
		h.groups++
	}
	return &h
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch level {
	case slog.LevelDebug:
		return h.logger.Level <= DebugLevel
	case slog.LevelInfo:
		return h.logger.Level <= InfoLevel
	case slog.LevelWarn:
		return h.logger.Level <= WarnLevel
	case slog.LevelError:
		return h.logger.Level <= ErrorLevel
	}
	return false
}

func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	var e *Entry
	switch r.Level {
	case slog.LevelDebug:
		e = h.logger.Debug()
	case slog.LevelInfo:
		e = h.logger.Info()
	case slog.LevelWarn:
		e = h.logger.Warn()
	case slog.LevelError:
		e = h.logger.Error()
	default:
		e = h.logger.Log()
	}

	if h.caller != 0 {
		e.caller(1, r.PC, h.caller < 0)
	}

	// msg
	e = e.Str("message", r.Message)

	// with
	if b := h.entry.buf; len(b) != 0 {
		e = e.Context(b)
	}
	i := len(e.buf)

	// attrs
	r.Attrs(func(attr slog.Attr) bool {
		e = slogAttrEval(e, attr)
		return true
	})

	lastindex := func(buf []byte) int {
		for i := len(buf) - 3; i >= 1; i-- {
			if buf[i] == '"' && (buf[i-1] == ',' || buf[i-1] == '{') {
				return i
			}
		}
		return -1
	}

	// group attrs
	if h.grouping {
		if r.NumAttrs() > 0 {
			e.buf[i] = '{'
		} else if i = lastindex(e.buf); i > 0 {
			e.buf = e.buf[:i-1]
			h.groups--
			for e.buf[len(e.buf)-1] == ':' {
				if i = lastindex(e.buf); i > 0 {
					e.buf = e.buf[:i-1]
					h.groups--
				}
			}
		} else {
			e.buf = append(e.buf, '{')
		}
	}

	// brackets closing
	switch h.groups {
	case 0:
		break
	case 1:
		e.buf = append(e.buf, '}')
	case 2:
		e.buf = append(e.buf, '}', '}')
	case 3:
		e.buf = append(e.buf, '}', '}', '}')
	case 4:
		e.buf = append(e.buf, '}', '}', '}', '}')
	default:
		for i := 0; i < h.groups; i++ {
			e.buf = append(e.buf, '}')
		}
	}

	e.Msg("")
	return nil
}

// Slog wraps the Logger to provide *slog.Logger
func (l *Logger) Slog() *slog.Logger {
	h := &slogHandler{
		logger: *l,
		caller: l.Caller,
	}

	h.logger.Caller = 0

	return slog.New(h)
}
