package main

import (
	"fmt"
	"os"

	"github.com/oarkflow/log"
)

func main() {
	// Example 1: Using the traditional MultiEntryWriter (no formatting control)
	w := []log.Writer{
		&log.HTTPWriter{
			URL: "http://localhost:3000/ingest/log",
		},
		&log.ConsoleWriter{
			ColorOutput: true,
		},
	}
	writer := log.MultiEntryWriter(w)
	logr := log.Logger{
		Level:     log.InfoLevel,
		Caller:    1,
		TimeField: "timestamp",
		Writer:    &writer,
	}
	logger := log.With(&logr).Str("resource", "pipelines").Copy()
	logger2 := log.With(&logger).Str("action", "create").Copy()
	logger2.Info().Str("loggable_type", "pipeline").Int("loggable_id", 42).Msg("Create pipeline")

	// Example 2: Using FormattedMultiLevelWriter with different formatters
	formattedWriter := &log.FormattedMultiLevelWriter{
		InfoWriter: &log.ConsoleWriter{
			Writer:      os.Stdout,
			ColorOutput: false,
		},
		InfoFormatter: &log.HumanReadableFormatter{
			ShowTimestamp: true,
			ShowLevel:     true,
			ShowCaller:    false,
		},
		WarnWriter: log.IOWriter{os.Stderr},
		ErrorWriter: &log.HTTPWriter{
			URL: "http://localhost:3000/errors",
		},
	}

	// Create a logger with the formatted writer
	formattedLogger := log.Logger{
		Level:  log.InfoLevel,
		Writer: formattedWriter,
	}

	// Test different log levels to see different formatters in action
	formattedLogger.Info().Str("service", "example").Msg("This will use human-readable format")
	formattedLogger.Warn().Str("warning", "test").Msg("This will use JSON format")
	formattedLogger.Error().Str("error", "example").Msg("This will be sent as JSON to HTTP endpoint")

	// Example 3: Using template-based formatting
	templateWriter := &log.FormattedMultiLevelWriter{
		InfoWriter:     log.IOWriter{os.Stdout},
		InfoFormatter:  log.DefaultTemplateFormatter("{{time}} [{{level}}] {{msg}}"),
		WarnWriter:     log.IOWriter{os.Stdout},
		WarnFormatter:  log.DefaultTemplateFormatter("{{time}} [{{level}}] {{msg}}{{if error}} - Error: {{error}}{{end}}"),
		ErrorWriter:    log.IOWriter{os.Stdout},
		ErrorFormatter: log.DefaultTemplateFormatter("{{time}} [{{level}}] {{msg}}{{if error}} - Error: {{error}}{{end}}"),
	}

	templateLogger := log.Logger{
		Level:  log.InfoLevel,
		Writer: templateWriter,
	}

	// Test template formatting
	templateLogger.Info().Str("user", "john").Msg("User logged in")
	templateLogger.Warn().Str("error", "connection timeout").Msg("Connection issue occurred")

	// Test conditional formatting with error
	templateLogger.Error().Err(fmt.Errorf("database connection failed")).Msg("Database operation failed")

	// Debug: Test with explicit error field
	templateLogger.Error().Str("error", "database connection failed").Msg("Database operation failed")
}

// Output:
// {"date":"2024-05-11","level":"info","attr1":"12","attr2":"1245","foo":"bar","number":42,"trace_id":"1789189541429514240","host_platform":"localhost","message":"hi, phuslog"}
// {"date":"2024-05-11","level":"error","attr1":"12","attr2":"1245","message":"foo=bar number=42 error=an error","trace_id":"1789189541555343360","host_platform":"localhost"}
