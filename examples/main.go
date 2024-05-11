package main

import (
	"os"

	"github.com/oarkflow/log"
)

func main() {
	w := []log.Writer{
		&log.ConsoleWriter{ColorOutput: true, EndWithMessage: true},
		&log.IOWriter{Writer: os.Stdout},
	}
	writer := log.MultiEntryWriter(w)
	logr := log.Logger{
		Level:      log.InfoLevel,
		Caller:     1,
		TimeField:  "date",
		TimeFormat: "2006-01-02",
		Writer:     &writer,
	}
	logger := log.With(&logr).Str("attr1", "12").Copy()
	logger2 := log.With(&logger).Str("attr2", "1245").Copy()
	logger2.Info().Str("foo", "bar").Int("number", 42).Msg("hi, phuslog")
	logger2.Error().Msgf("foo=%s number=%d error=%+v", "bar", 42, "an error")
}

// Output:
// {"date":"2024-05-11","level":"info","attr1":"12","attr2":"1245","foo":"bar","number":42,"trace_id":"1789189541429514240","host_platform":"localhost","message":"hi, phuslog"}
// {"date":"2024-05-11","level":"error","attr1":"12","attr2":"1245","message":"foo=bar number=42 error=an error","trace_id":"1789189541555343360","host_platform":"localhost"}
