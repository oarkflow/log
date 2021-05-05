package log

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sujit-baniya/xid"
)

type Config struct {
	Logger    *Logger
	LogWriter Writer
	RequestID func() string
}

//Middleware request_id + logger + recover for request traceability
func Middleware(config Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		if strings.Contains(c.Path(), "favicon") {
			return c.Next()
		}
		rid := c.Get(fiber.HeaderXRequestID)
		if config.RequestID == nil {
			config.RequestID = func() string {
				return xid.New().String()
			}
		}
		if rid == "" {
			rid = config.RequestID()
			c.Set(fiber.HeaderXRequestID, rid)
		}
		nextHandler := c.Next()
		if c.Route().Path == "/" && c.Path() != c.Route().Path {
			return nextHandler
		}
		if config.Logger == nil {
			config.Logger = &Logger{
				TimeField:  "timestamp",
				TimeFormat: "2006-01-02 15:04:05",
			}
		}
		if config.LogWriter != nil {
			config.Logger.Writer = config.LogWriter
		}
		ip := c.IP()
		curIP := c.Locals("ip")
		if curIP != nil {
			ip = curIP.(string)
		}
		logging := NewContext(nil).
			Str("request_id", rid).
			Str("remote_ip", ip).
			Str("method", c.Method()).
			Str("host", c.Hostname()).
			Str("path", c.Path()).
			Str("protocol", c.Protocol()).
			Int("status", c.Response().StatusCode()).
			Str("latency", fmt.Sprintf("%s", time.Since(start))).
			Str("ua", c.Get(fiber.HeaderUserAgent))

		if nextHandler != nil {
			logging.Str("error", nextHandler.Error())
		}
		ctx := logging.Value()
		switch {
		case c.Response().StatusCode() >= 500:
			config.Logger.Error().Context(ctx).Msg("server error")
		case c.Response().StatusCode() >= 400:
			config.Logger.Error().Context(ctx).Msg("client error")
		case c.Response().StatusCode() >= 300:
			config.Logger.Warn().Context(ctx).Msg("redirect")
		case c.Response().StatusCode() >= 200:
			config.Logger.Info().Context(ctx).Msg("success")
		case c.Response().StatusCode() >= 100:
			config.Logger.Info().Context(ctx).Msg("informative")
		default:
			config.Logger.Warn().Context(ctx).Msg("unknown status")
		}
		return nextHandler
	}
}
