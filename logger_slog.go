//go:build go1.21
// +build go1.21

package log

import (
	"context"
	"log/slog"
	"sync"
)

type slogGroup struct {
	name  string
	attrs []any // *slog.Attr or *slogGroup
}

func lastGroup(attrs []any) *slogGroup {
	if len(attrs) == 0 {
		return nil
	}
	group, _ := attrs[len(attrs)-1].(*slogGroup)
	return group
}

func (group *slogGroup) WithAttrs(attrs []slog.Attr) {
	if g := lastGroup(group.attrs); g != nil {
		g.WithAttrs(attrs)
	} else {
		for _, attr := range attrs {
			group.attrs = append(group.attrs, &attr)
		}
	}
}

func (group *slogGroup) WithGroup(name string) {
	if g := lastGroup(group.attrs); g != nil {
		g.WithGroup(name)
	} else {
		group.attrs = append(group.attrs, &slogGroup{name: name})
	}
}

func (group *slogGroup) Output(e *Entry) *Entry {
	b := bbpool.Get().(*bb)
	defer bbpool.Put(b)
	for _, v := range group.attrs {
		if g, ok := v.(*slogGroup); ok && g.name != "" {
			e.Dict(g.name, g.Output(NewContext(b.B[:0])).Value())
		} else if attr, ok := v.(*slog.Attr); ok {
			e = e.Any(attr.Key, attr.Value)
		}
	}
	return e
}

type slogHandler struct {
	Logger

	group   slogGroup
	once    sync.Once
	context Context
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if group := lastGroup(h.group.attrs); group != nil {
		group.WithAttrs(attrs)
	} else {
		for _, attr := range attrs {
			h.group.attrs = append(h.group.attrs, &attr)
		}
	}
	return h
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if group := lastGroup(h.group.attrs); group != nil {
		group.WithGroup(name)
	} else {
		h.group.attrs = append(h.group.attrs, &slogGroup{name: name})
	}
	return h
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch level {
	case slog.LevelDebug:
		return h.Logger.Level <= DebugLevel
	case slog.LevelInfo:
		return h.Logger.Level <= InfoLevel
	case slog.LevelWarn:
		return h.Logger.Level <= WarnLevel
	case slog.LevelError:
		return h.Logger.Level <= ErrorLevel
	}
	return false
}

func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	var e *Entry
	switch r.Level {
	case slog.LevelDebug:
		e = h.Logger.Debug()
	case slog.LevelInfo:
		e = h.Logger.Info()
	case slog.LevelWarn:
		e = h.Logger.Warn()
	case slog.LevelError:
		e = h.Logger.Error()
	default:
		e = h.Logger.Log()
	}
	h.once.Do(func() {
		if len(h.group.attrs) != 0 {
			h.context = h.group.Output(NewContext(nil)).Value()
		}
	})
	if h.context != nil {
		e = e.Context(h.context)
	}
	r.Attrs(func(attr slog.Attr) bool {
		e = e.Any(attr.Key, attr.Value)
		return true
	})
	e.Msg(r.Message)
	return nil
}

// Slog wraps the Logger to provide *slog.Logger
func (l *Logger) Slog() *slog.Logger {
	logger := *l
	switch {
	case logger.Caller > 0:
		logger.Caller += 3
	case logger.Caller < 0:
		logger.Caller -= 3
	}
	return slog.New(&slogHandler{Logger: logger})
}
