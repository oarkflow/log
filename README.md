# Structured Logging Made Easy

[![go.dev][pkg-img]][pkg] [![goreport][report-img]][report] [![build][build-img]][build] [![coverage][cov-img]][cov] ![stability-stable][stability-img]

## Features

* No Dependencies
* Intuitive Interfaces
* Consistent Writers
    - `IOWriter`, *io.Writer wrapper*
    - `FileWriter`, *rotating & effective*
    - `ConsoleWriter`, *colorful & formatting*
    - `MultiWriter`, *multiple level dispatch*
    - `SyslogWriter`, *syslog server logging*
    - `JournalWriter`, *linux systemd logging*
    - `EventlogWriter`, *windows system event*
    - `AsyncWriter`, *asynchronously writing*
* Third-party Logger Interceptor
    - `Logger.Std`, *(std)log*
    - `Logger.Grpc`, *grpclog.LoggerV2*
    - `Logger.Logr`, *logr.Logger*
    - `Logger.Sugar`, *zap.SugaredLogger*
* Useful utility functions
    - `Goid()`, *current goroutine id*
    - `NewXID()`, *create a tracing id*
    - `Fastrandn(n uint32)`, *fast pseudorandom uint32 in [0,n)*
    - `IsTerminal(fd uintptr)`, *isatty for golang*
    - `Printf(fmt string, a ...interface{})`, *printf logging*
* High Performance
    - [Significantly faster][high-performance] than all other json loggers.

## Interfaces

### Logger
```go
// DefaultLogger is the global logger.
var DefaultLogger = Logger{
	Level:      DebugLevel,
	Caller:     0,
	TimeField:  "",
	TimeFormat: "",
	Writer:     &IOWriter{os.Stderr},
}

// A Logger represents an active logging object that generates lines of JSON output to an io.Writer.
type Logger struct {
	// Level defines log levels.
	Level Level

	// Caller determines if adds the file:line of the "caller" key.
	Caller int

	// TimeField defines the time filed name in output.  It uses "time" in if empty.
	TimeField string

	// TimeFormat specifies the time format in output. It uses RFC3339 with millisecond if empty.
	// If set with `TimeFormatUnix`, `TimeFormatUnixMs`, times are formated as UNIX timestamp.
	TimeFormat string

	// Writer specifies the writer of output. It uses a wrapped os.Stderr Writer in if empty.
	Writer Writer
}

// Writer defines an entry writer interface.
type Writer interface {
	WriteEntry(*Entry) (int, error)
}
```

### FileWriter & ConsoleWriter
```go
// FileWriter is an Writer that writes to the specified filename.
type FileWriter struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.
	Filename string

	// FileMode represents the file's mode and permission bits.  The default
	// mode is 0644
	FileMode os.FileMode

	// MaxSize is the maximum size in bytes of the log file before it gets rotated.
	MaxSize int64

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files
	MaxBackups int

	// CleanBackups specifies an optional cleanup function of log backups after rotation,
	// if not set, the default behavior is to delete more than MaxBackups log files.
	CleanBackups func(filename string, maxBackups int, matches []os.FileInfo)

	// TimeFormat specifies the time format of filename. It uses `2006-01-02T15-04-05` as default format.
	// If set with `TimeFormatUnix`, `TimeFormatUnixMs`, times are formated as UNIX timestamp.
	TimeFormat string

	// LocalTime determines if the time used for formatting the timestamps in
	// log files is the computer's local time.  The default is to use UTC time.
	LocalTime bool

	// HostName determines if the hostname used for formatting in log files.
	HostName bool

	// ProcessID determines if the pid used for formatting in log files.
	ProcessID bool

	// EnsureFolder ensures the file directory creation before writing.
	EnsureFolder bool
}

// ConsoleWriter parses the JSON input and writes it in an
// (optionally) colorized, human-friendly format to Writer.
//
// Default output format:
//     {Time} {Level} {Goid} {Caller} > {Message} {Key}={Value} {Key}={Value}
type ConsoleWriter struct {
	// ColorOutput determines if used colorized output.
	ColorOutput bool

	// QuoteString determines if quoting string values.
	QuoteString bool

	// EndWithMessage determines if output message in the end of line.
	EndWithMessage bool

	// Formatter specifies an optional text formatter for creating a customized output,
	// If it is set, ColorOutput, QuoteString and EndWithMessage will be ignored.
	Formatter func(w io.Writer, args *FormatterArgs) (n int, err error)

	// Writer is the output destination. using os.Stderr if empty.
	Writer io.Writer
}
```
> Note: FileWriter/ConsoleWriter implements log.Writer and io.Writer interfaces both.

## Getting Started

### Simple Logging Example

A out of box example. [![playground][play-simple-img]][play-simple]
```go
package main

import (
	"github.com/phuslu/log"
)

func main() {
	log.Printf("Hello, %s", "世界")
	log.Info().Str("foo", "bar").Int("number", 42).Msg("hi, phuslog")
	log.Error().Str("foo", "bar").Int("number", 42).Msgf("oops, %s", "phuslog")
}

// Output:
//   {"time":"2020-03-22T09:58:41.828Z","message":"Hello, 世界"}
//   {"time":"2020-03-22T09:58:41.828Z","level":"info","foo":"bar","number":42,"message":"hi, phuslog"}
//   {"time":"2020-03-22T09:58:41.828Z","level":"error","foo":"bar","number":42,"message":"oops, phuslog"}
```
> Note: By default log writes to `os.Stderr`

### Customize the configuration and formatting:

To customize logger filed name and format. [![playground][play-customize-img]][play-customize]
```go
log.DefaultLogger = log.Logger{
	Level:      log.InfoLevel,
	Caller:     1,
	TimeField:  "date",
	TimeFormat: "2006-01-02",
	Writer:     &log.IOWriter{os.Stdout},
}
log.Info().Str("foo", "bar").Msgf("hello %s", "world")

// Output: {"date":"2019-07-04","level":"info","caller":"prog.go:16","foo":"bar","message":"hello world"}
```

### Rotating File Writer

To log to a rotating file, use `FileWriter`. [![playground][play-file-img]][play-file]
```go
package main

import (
	"time"

	"github.com/phuslu/log"
	"github.com/robfig/cron/v3"
)

func main() {
	logger := log.Logger{
		Level:      log.ParseLevel("info"),
		Writer:     &log.FileWriter{
			Filename:     "logs/main.log",
			FileMode:     0600,
			MaxSize:      50*1024*1024,
			MaxBackups:   7,
			EnsureFolder: true,
			LocalTime:    false,
		},
	}

	runner := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))
	runner.AddFunc("0 0 * * * *", func() { logger.Writer.(*log.FileWriter).Rotate() })
	go runner.Run()

	for {
		time.Sleep(time.Second)
		logger.Info().Msg("hello world")
	}
}
```

### Pretty Console Writer

To log a human-friendly, colorized output, use `ConsoleWriter`. [![playground][play-pretty-img]][play-pretty]

```go
if log.IsTerminal(os.Stderr.Fd()) {
	log.DefaultLogger = log.Logger{
		Caller: 1,
		Writer: &log.ConsoleWriter{
			ColorOutput:    true,
			QuoteString:    true,
			EndWithMessage: true,
		},
	}
}

log.Debug().Int("everything", 42).Str("foo", "bar").Msg("hello world")
log.Info().Int("everything", 42).Str("foo", "bar").Msg("hello world")
log.Warn().Int("everything", 42).Str("foo", "bar").Msg("hello world")
log.Error().Err(errors.New("an error")).Msg("hello world")
```
![Pretty logging][pretty-img]
> Note: pretty logging also works on windows console

### Formatting Console Writer

To log with user-defined format(e.g. glog), using `ConsoleWriter.Formatter`. [![playground][play-formatting-img]][play-formatting]

```go
package main

import (
	"io"
	"fmt"
	"github.com/phuslu/log"
)

var glog = (&log.Logger{
	Level:      log.InfoLevel,
	Caller:     1,
	TimeFormat: "0102 15:04:05.999999",
	Writer: &log.ConsoleWriter{
		Formatter: func (w io.Writer, a *log.FormatterArgs) (int, error) {
			return fmt.Fprintf(w, "%c%s %s %s] %s\n%s",
				a.Level[0]-32, a.Time, a.Goid, a.Caller, a.Message, a.Stack)
		},
	},
}).Sugar(nil)

func main() {
	glog.Infof("hello glog %s", "Info")
	glog.Warnf("hello glog %s", "Warn")
	glog.Errorf("hello glog %s", "Error")
	glog.Fatalf("hello glog %s", "Fatal")
}

// Output:
// I0725 09:59:57.503246 19 console_test.go:183] hello glog Info
// W0725 09:59:57.504247 19 console_test.go:184] hello glog Warn
// E0725 09:59:57.504247 19 console_test.go:185] hello glog Error
```

### Multiple Dispatching Writer

To log to different writers by different levels, use `MultiWriter`.

```go
log.DefaultLogger.Writer = &log.MultiWriter{
	InfoWriter:    &log.FileWriter{Filename: "main.INFO", MaxSize: 100<<20},
	WarnWriter:    &log.FileWriter{Filename: "main.WARNING", MaxSize: 100<<20},
	ErrorWriter:   &log.FileWriter{Filename: "main.ERROR", MaxSize: 100<<20},
	ConsoleWriter: &log.ConsoleWriter{ColorOutput: true},
	ConsoleLevel:  log.ErrorLevel,
}
log.Info().Int("number", 42).Str("foo", "bar").Msg("a info log")
log.Warn().Int("number", 42).Str("foo", "bar").Msg("a warn log")
log.Error().Int("number", 42).Str("foo", "bar").Msg("a error log")
```

### SyslogWriter & JournalWriter & EventlogWriter

To log to a syslog server, using `SyslogWriter`.

```go
log.DefaultLogger.Writer = &log.SyslogWriter{
	Network : "unixgram",                     // "tcp"
	Address : "/run/systemd/journal/syslog",  // "192.168.0.2:601"
	Tag     : "",
	Marker  : "@cee:",
	Dial    : net.Dial,
}
log.Info().Msg("hi")

// Output: <6>Oct 5 16:25:38 [237]: @cee:{"time":"2020-10-05T16:25:38.026Z","level":"info","message":"hi"}
```

To log to linux systemd journald, using `JournalWriter`.

```go
log.DefaultLogger.Writer = &log.JournalWriter{}
log.Info().Int("number", 42).Str("foo", "bar").Msg("hello world")
```

To log to windows system event, using `EventlogWriter`.

```go
log.DefaultLogger.Writer = &log.EventlogWriter{
	Source: ".NET Runtime",
	ID:     1000,
}
log.Info().Int("number", 42).Str("foo", "bar").Msg("hello world")
```

### AsyncWriter

To logging asynchronously for performance stability, use `AsyncWriter`.

```go
logger := log.Logger{
	Level:  log.InfoLevel,
	Writer: &log.AsyncWriter{
		ChannelSize: 100,
		Writer:      &log.FileWriter{
			Filename:   "main.log",
			FileMode:   0600,
			MaxSize:    50*1024*1024,
			MaxBackups: 7,
			LocalTime:  false,
		},
	},
}

logger.Info().Int("number", 42).Str("foo", "bar").Msg("a async info log")
logger.Warn().Int("number", 42).Str("foo", "bar").Msg("a async warn log")
logger.Writer.(io.Closer).Close()
```

> Note: To flush data and quit safely, call `AsyncWriter.Close()` explicitly.

### Sugar Logger

In contexts where performance is nice, but not critical, use the `SugaredLogger`. It's 20% slower than `Logger` but still faster than other structured logging packages [![playground][play-sugar-img]][play-sugar]

```go
package main

import (
	"github.com/phuslu/log"
)

func main() {
	sugar := log.DefaultLogger.Sugar(log.NewContext(nil).Str("tag", "hi suagr").Value())
	sugar.Infof("hello %s", "世界")
	sugar.Infow("i am a leading message", "foo", "bar", "number", 42)

	sugar = sugar.Level(log.ErrorLevel)
	sugar.Printf("hello %s", "世界")
	sugar.Log("number", 42, "a_key", "a_value", "message", "a suagr message")
}
```

### StdLog & Logr & Grpc Interceptor

Using wrapped loggers for stdlog/grpc/logr. [![playground][play-interceptor-img]][play-interceptor]

```go
package main

import (
	stdLog "log"
	"github.com/go-logr/logr"
	"github.com/phuslu/log"
	"google.golang.org/grpc/grpclog"
)

func main() {
	ctx := log.NewContext(nil).Str("tag", "hi log").Value()

	var stdlog *stdLog.Logger = log.DefaultLogger.Std(log.InfoLevel, ctx, "prefix ", stdLog.LstdFlags)
	stdlog.Print("hello from stdlog Print")
	stdlog.Println("hello from stdlog Println")
	stdlog.Printf("hello from stdlog %s", "Printf")

	var grpclog grpclog.LoggerV2 = log.DefaultLogger.Grpc(ctx)
	grpclog.Infof("hello %s", "grpclog Infof message")
	grpclog.Errorf("hello %s", "grpclog Errorf message")

	var logrLog logr.Logger = log.DefaultLogger.Logr(ctx)
	logrLog = logrLog.WithName("a_named_logger").WithValues("a_key", "a_value")
	logrLog.Info("hello", "foo", "bar", "number", 42)
	logrLog.Error(errors.New("this is a error"), "hello", "foo", "bar", "number", 42)
}
```

### Contextual Fields

To add preserved `key:value` pairs to each entry, use `NewContext`. [![playground][play-context-img]][play-context]

```go
ctx := log.NewContext(nil).Str("ctx_str", "a ctx str").Value()

logger := log.Logger{Level: log.InfoLevel}
logger.Debug().Context(ctx).Int("no0", 0).Msg("zero")
logger.Info().Context(ctx).Int("no1", 1).Msg("first")
logger.Info().Context(ctx).Int("no2", 2).Msg("second")

// Output:
//   {"time":"2020-07-12T05:03:43.949Z","level":"info","ctx_str":"a ctx str","no1":1,"message":"first"}
//   {"time":"2020-07-12T05:03:43.949Z","level":"info","ctx_str":"a ctx str","no2":2,"message":"second"}
```

### High Performance

A quick and simple benchmark with zap/zerolog, which runs on [github actions][benchmark]:

```go
// go test -v -cpu=4 -run=none -bench=. -benchtime=10s -benchmem log_test.go
package main

import (
	"io/ioutil"
	"testing"

	"github.com/phuslu/log"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var fakeMessage = "Test logging, but use a somewhat realistic message length. "

func BenchmarkZap(b *testing.B) {
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(ioutil.Discard),
		zapcore.InfoLevel,
	))
	for i := 0; i < b.N; i++ {
		logger.Info(fakeMessage, zap.String("foo", "bar"), zap.Int("int", 42))
	}
}

func BenchmarkZeroLog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard).With().Timestamp().Logger()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("int", 42).Msg(fakeMessage)
	}
}

func BenchmarkPhusLog(b *testing.B) {
	logger := log.Logger{
		TimeFormat: "", // uses rfc3339 by default
		Writer:     log.IOWriter{ioutil.Discard},
	}
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("int", 42).Msg(fakeMessage)
	}
}
```
A Performance result as below, for daily benchmark results see [github actions][benchmark]
```

BenchmarkZap-4       	 9126241	      1293 ns/op	     128 B/op	       1 allocs/op
BenchmarkZeroLog-4   	18187580	       638 ns/op	       0 B/op	       0 allocs/op
BenchmarkPhusLog-4   	42004888	       287 ns/op	       0 B/op	       0 allocs/op
```

## A Real World Example

The example starts a geoip http server which supports change log level dynamically
```go
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/naoina/toml"
	"github.com/phuslu/iploc"
	"github.com/phuslu/log"
)

type Config struct {
	Listen struct {
		Tcp string
	}
	Log struct {
		Level   string
		Maxsize int64
		Backups int
	}
}

type Handler struct {
	Config       *Config
	AccessLogger log.Logger
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqID := log.NewXID()
	remoteIP, _, _ := net.SplitHostPort(req.RemoteAddr)
	geo := iploc.Country(net.ParseIP(remoteIP))

	h.AccessLogger.Log().
		Xid("req_id", reqID).
		Str("host", req.Host).
		Bytes("geo", geo).
		Str("remote_ip", remoteIP).
		Str("request_uri", req.RequestURI).
		Str("user_agent", req.UserAgent()).
		Str("referer", req.Referer()).
		Msg("access log")

	switch req.RequestURI {
	case "/debug", "/info", "/warn", "/error":
		log.DefaultLogger.SetLevel(log.ParseLevel(req.RequestURI[1:]))
	default:
		fmt.Fprintf(rw, `{"req_id":"%s","ip":"%s","geo":"%s"}`, reqID, remoteIP, geo)
	}
}

func main() {
	config := new(Config)
	err := toml.Unmarshal([]byte(`
[listen]
tcp = ":8080"

[log]
level = "debug"
backups = 5
maxsize = 1073741824
        `), config)
	if err != nil {
		log.Fatal().Msgf("toml.Unmarshal error: %+v", err)
	}

	handler := &Handler{
		Config: config,
		AccessLogger: log.Logger{
			Writer: &log.FileWriter{
				Filename:   "access.log",
				MaxSize:    config.Log.Maxsize,
				MaxBackups: config.Log.Backups,
				LocalTime:  true,
			},
		},
	}

	if log.IsTerminal(os.Stderr.Fd()) {
		log.DefaultLogger = log.Logger{
			Level:      log.ParseLevel(config.Log.Level),
			Caller:     1,
			TimeFormat: "15:04:05",
			Writer: &log.ConsoleWriter{
				ColorOutput:    true,
				EndWithMessage: true,
			},
		}
		handler.AccessLogger = log.DefaultLogger
	} else {
		log.DefaultLogger = log.Logger{
			Level: log.ParseLevel(config.Log.Level),
			Writer: &log.FileWriter{
				Filename:   "main.log",
				MaxSize:    config.Log.Maxsize,
				MaxBackups: config.Log.Backups,
				LocalTime:  true,
			},
		}
	}

	server := &http.Server{
		Addr:     config.Listen.Tcp,
		ErrorLog: log.DefaultLogger.Std(log.ErrorLevel, nil, "", 0),
		Handler:  handler,
	}

	log.Fatal().Err(server.ListenAndServe()).Msg("listen failed")
}
```

### Acknowledgment
This log is heavily inspired by [zerolog][zerolog], [glog][glog], [quicktemplate][quicktemplate], [gjson][gjson], [zap][zap] and [lumberjack][lumberjack].

[pkg-img]: http://img.shields.io/badge/godoc-reference-5272B4.svg
[pkg]: https://godoc.org/github.com/phuslu/log
[report-img]: https://goreportcard.com/badge/github.com/phuslu/log
[report]: https://goreportcard.com/report/github.com/phuslu/log
[build-img]: https://github.com/phuslu/log/workflows/build/badge.svg
[build]: https://github.com/phuslu/log/actions
[cov-img]: http://gocover.io/_badge/github.com/phuslu/log
[cov]: https://gocover.io/github.com/phuslu/log
[stability-img]: https://img.shields.io/badge/stability-stable-green.svg
[high-performance]: https://github.com/phuslu/log#high-performance
[play-simple-img]: https://img.shields.io/badge/playground-NGV25aBKmYH-29BEB0?style=flat&logo=go
[play-simple]: https://play.golang.org/p/NGV25aBKmYH
[play-customize-img]: https://img.shields.io/badge/playground-emTsJJKUGXZ-29BEB0?style=flat&logo=go
[play-customize]: https://play.golang.org/p/emTsJJKUGXZ
[play-file-img]: https://img.shields.io/badge/playground-nS--ILxFyhHM-29BEB0?style=flat&logo=go
[play-file]: https://play.golang.org/p/nS-ILxFyhHM
[play-pretty-img]: https://img.shields.io/badge/playground-CD1LClgEvS4-29BEB0?style=flat&logo=go
[play-pretty]: https://play.golang.org/p/CD1LClgEvS4
[pretty-img]: https://user-images.githubusercontent.com/195836/90043818-37d99900-dcff-11ea-9f93-7de9ce8b7316.png
[play-formatting-img]: https://img.shields.io/badge/playground-8ScRKLIrehG-29BEB0?style=flat&logo=go
[play-formatting]: https://play.golang.org/p/8ScRKLIrehG
[play-context-img]: https://img.shields.io/badge/playground-oAVAo302faf-29BEB0?style=flat&logo=go
[play-context]: https://play.golang.org/p/oAVAo302faf
[play-sugar-img]: https://img.shields.io/badge/playground-iGfD_wOcA6c-29BEB0?style=flat&logo=go
[play-sugar]: https://play.golang.org/p/iGfD_wOcA6c
[play-interceptor]: https://play.golang.org/p/upmVP5cO62Y
[play-interceptor-img]: https://img.shields.io/badge/playground-upmVP5cO62Y-29BEB0?style=flat&logo=go
[benchmark]: https://github.com/phuslu/log/actions?query=workflow%3Abenchmark
[zerolog]: https://github.com/rs/zerolog
[glog]: https://github.com/golang/glog
[quicktemplate]: https://github.com/valyala/quicktemplate
[gjson]: https://github.com/tidwall/gjson
[zap]: https://github.com/uber-go/zap
[lumberjack]: https://github.com/natefinch/lumberjack
