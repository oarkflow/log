package main

import (
	"github.com/oarkflow/log"
)

func main() {
	log.DefaultLogger.TimeField = "date"
	log.DefaultLogger.TimeFormat = "2006-01-02"
	logger := log.With(&log.DefaultLogger).Str("attr1", "12").Copy()
	logger2 := log.With(&logger).Str("attr2", "1245").Copy()
	logger2.Info().Str("foo", "bar").Int("number", 42).Msg("hi, phuslog")
	logger2.Error().Msgf("foo=%s number=%d error=%+v", "bar", 42, "an error")
}

// Output:
// {"date":"2024-05-11","level":"info","attr1":"12","attr2":"1245","foo":"bar","number":42,"trace_id":"1789189541429514240","host_platform":"localhost","message":"hi, phuslog"}
// {"date":"2024-05-11","level":"error","attr1":"12","attr2":"1245","message":"foo=bar number=42 error=an error","trace_id":"1789189541555343360","host_platform":"localhost"}
