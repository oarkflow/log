package log

import (
	"io"
)

// MultiWriter is an alias for MultiLevelWriter
type MultiWriter = MultiLevelWriter

// MultiLevelWriter is an Writer that log to different writers by different levels
type MultiLevelWriter struct {
	// InfoWriter specifies all the level logs writes to
	InfoWriter Writer

	// WarnWriter specifies the level greater than or equal to WarnLevel writes to
	WarnWriter Writer

	// WarnWriter specifies the level greater than or equal to ErrorLevel writes to
	ErrorWriter Writer

	// ConsoleWriter specifies the console writer
	ConsoleWriter Writer

	// ConsoleLevel specifies the level greater than or equal to it also writes to
	ConsoleLevel Level
}

// Close implements io.Closer, and closes the underlying LeveledWriter.
func (w *MultiLevelWriter) Close() (err error) {
	for _, writer := range []Writer{
		w.InfoWriter,
		w.WarnWriter,
		w.ErrorWriter,
		w.ConsoleWriter,
	} {
		if writer == nil {
			continue
		}
		if closer, ok := writer.(io.Closer); ok {
			if err1 := closer.Close(); err1 != nil {
				err = err1
			}
		}
	}
	return
}

// WriteEntry implements entryWriter.
func (w *MultiLevelWriter) WriteEntry(e *Entry) (n int, err error) {
	var err1 error
	switch e.Level {
	case noLevel, PanicLevel, FatalLevel, ErrorLevel:
		if w.ErrorWriter != nil {
			n, err1 = w.ErrorWriter.WriteEntry(e)
			if err1 != nil && err == nil { //nolint:nilness
				err = err1
			}
		}
		fallthrough
	case WarnLevel:
		if w.WarnWriter != nil {
			n, err1 = w.WarnWriter.WriteEntry(e)
			if err1 != nil && err == nil { //nolint:nilness
				err = err1
			}
		}
		fallthrough
	default:
		if w.InfoWriter != nil {
			n, err1 = w.InfoWriter.WriteEntry(e)
			if err1 != nil && err == nil { //nolint:nilness
				err = err1
			}
		}
	}

	if w.ConsoleWriter != nil && e.Level >= w.ConsoleLevel {
		_, _ = w.ConsoleWriter.WriteEntry(e)
	}

	return
}

var _ Writer = (*MultiLevelWriter)(nil)

// FormattedMultiLevelWriter is a MultiLevelWriter that supports different formatters for different writers
type FormattedMultiLevelWriter struct {
	// InfoWriter specifies all the level logs writes to
	InfoWriter Writer
	// InfoFormatter specifies the formatter for InfoWriter
	InfoFormatter Formatter

	// WarnWriter specifies the level greater than or equal to WarnLevel writes to
	WarnWriter Writer
	// WarnFormatter specifies the formatter for WarnWriter
	WarnFormatter Formatter

	// ErrorWriter specifies the level greater than or equal to ErrorLevel writes to
	ErrorWriter Writer
	// ErrorFormatter specifies the formatter for ErrorWriter
	ErrorFormatter Formatter

	// ConsoleWriter specifies the console writer
	ConsoleWriter Writer
	// ConsoleFormatter specifies the formatter for ConsoleWriter
	ConsoleFormatter Formatter

	// ConsoleLevel specifies the level greater than or equal to it also writes to
	ConsoleLevel Level
}

// SetInfoFormatter sets the formatter for the InfoWriter
func (w *FormattedMultiLevelWriter) SetInfoFormatter(formatter Formatter) {
	w.InfoFormatter = formatter
}

// SetWarnFormatter sets the formatter for the WarnWriter
func (w *FormattedMultiLevelWriter) SetWarnFormatter(formatter Formatter) {
	w.WarnFormatter = formatter
}

// SetErrorFormatter sets the formatter for the ErrorWriter
func (w *FormattedMultiLevelWriter) SetErrorFormatter(formatter Formatter) {
	w.ErrorFormatter = formatter
}

// SetConsoleFormatter sets the formatter for the ConsoleWriter
func (w *FormattedMultiLevelWriter) SetConsoleFormatter(formatter Formatter) {
	w.ConsoleFormatter = formatter
}

// Close implements io.Closer, and closes the underlying FormattedMultiLevelWriter.
func (w *FormattedMultiLevelWriter) Close() (err error) {
	for _, writer := range []Writer{
		w.InfoWriter,
		w.WarnWriter,
		w.ErrorWriter,
		w.ConsoleWriter,
	} {
		if writer == nil {
			continue
		}
		if closer, ok := writer.(io.Closer); ok {
			if err1 := closer.Close(); err1 != nil {
				err = err1
			}
		}
	}
	return
}

// WriteEntry implements Writer with per-writer formatting.
func (w *FormattedMultiLevelWriter) WriteEntry(e *Entry) (n int, err error) {
	// Helper function to write with formatter
	writeWithFormatter := func(writer Writer, formatter Formatter, entry *Entry) {
		if writer == nil {
			return
		}

		var formattedEntry []byte
		var formatErr error

		if formatter != nil {
			formattedEntry, formatErr = formatter.Format(entry)
			if formatErr != nil {
				// If formatting fails, fall back to original buffer
				_, _ = writer.WriteEntry(entry)
				return
			}
		} else {
			// No formatter, use original behavior
			_, _ = writer.WriteEntry(entry)
			return
		}

		// Write formatted data directly to the underlying io.Writer
		if ioWriter, ok := writer.(interface{ Write([]byte) (int, error) }); ok {
			ioWriter.Write(formattedEntry)
		} else {
			// Create a temporary entry with formatted data
			tempEntry := &Entry{
				buf:   formattedEntry,
				Level: entry.Level,
			}
			writer.WriteEntry(tempEntry)
		}
	}

	switch e.Level {
	case noLevel, PanicLevel, FatalLevel, ErrorLevel:
		writeWithFormatter(w.ErrorWriter, w.ErrorFormatter, e)
	case WarnLevel:
		writeWithFormatter(w.WarnWriter, w.WarnFormatter, e)
	default:
		writeWithFormatter(w.InfoWriter, w.InfoFormatter, e)
	}

	if w.ConsoleWriter != nil && e.Level >= w.ConsoleLevel {
		writeWithFormatter(w.ConsoleWriter, w.ConsoleFormatter, e)
	}

	return
}

var _ Writer = (*FormattedMultiLevelWriter)(nil)

// MultiEntryWriter is an array Writer that log to different writers
type MultiEntryWriter []Writer

// Close implements io.Closer, and closes the underlying MultiEntryWriter.
func (w *MultiEntryWriter) Close() (err error) {
	for _, writer := range *w {
		if closer, ok := writer.(io.Closer); ok {
			if err1 := closer.Close(); err1 != nil {
				err = err1
			}
		}
	}
	return
}

// WriteEntry implements entryWriter.
func (w *MultiEntryWriter) WriteEntry(e *Entry) (n int, err error) {
	var err1 error
	for _, writer := range *w {
		n, err1 = writer.WriteEntry(e)
		if err1 != nil && err == nil {
			err = err1
		}
	}
	return
}

var _ Writer = (*MultiEntryWriter)(nil)

// FormattedWriter extends Writer with formatting capabilities
type FormattedWriter interface {
	Writer
	SetFormatter(formatter Formatter)
}

// MultiIOWriter is an array io.Writer that log to different writers
type MultiIOWriter []io.Writer

// Close implements io.Closer, and closes the underlying MultiIOWriter.
func (w *MultiIOWriter) Close() (err error) {
	for _, writer := range *w {
		if closer, ok := writer.(io.Closer); ok {
			if err1 := closer.Close(); err1 != nil {
				err = err1
			}
		}
	}
	return
}

// WriteEntry implements entryWriter.
func (w *MultiIOWriter) WriteEntry(e *Entry) (n int, err error) {
	for _, writer := range *w {
		n, err = writer.Write(e.buf)
		if err != nil {
			return
		}
	}

	return
}
