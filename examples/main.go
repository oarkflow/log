package main

import (
	"github.com/oarkflow/log"
)

func main() {
	w := []log.Writer{
		&log.HTTPWriter{
			URL: "http://localhost:3000/ingest/log",
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
}

// Output:
// {"date":"2024-05-11","level":"info","attr1":"12","attr2":"1245","foo":"bar","number":42,"trace_id":"1789189541429514240","host_platform":"localhost","message":"hi, phuslog"}
// {"date":"2024-05-11","level":"error","attr1":"12","attr2":"1245","message":"foo=bar number=42 error=an error","trace_id":"1789189541555343360","host_platform":"localhost"}
