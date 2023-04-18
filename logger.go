package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type LoggerMiddleware struct {
	UTC       bool
	SkipPaths []string
	Level     logrus.Level
}

func (m LoggerMiddleware) Use(r *gin.Engine) {
	middleware := func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		url := c.Request.URL.RequestURI()

		c.Next()

		track := true
		for _, p := range m.SkipPaths {
			if strings.Contains(path, p) {
				track = false

				break
			}
		}

		for _, p := range m.SkipPaths {
			if strings.Contains(url, p) {
				track = false

				break
			}
		}

		if track {
			end := time.Now()
			latency := end.Sub(start)

			if m.UTC {
				end = end.UTC()
			}

			msg := "Request"
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}

			log := logrus.WithFields(logrus.Fields{
				"module":  "http",
				"method":  c.Request.Method,
				"path":    url,
				"status":  c.Writer.Status(),
				"latency": latency,
				"ip":      c.ClientIP(),
			})
			log.Logger.SetLevel(m.Level)

			switch {
			case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
				log.Warn(msg)

			case c.Writer.Status() >= http.StatusInternalServerError:
				log.Error(msg)

			default:
				log.Info(msg)
			}
		}
	}

	r.Use(middleware)
}

func NewLogger(utc bool, skipPaths []string, level logrus.Level) *LoggerMiddleware {
	return &LoggerMiddleware{
		UTC:       utc,
		SkipPaths: skipPaths,
		Level:     level,
	}
}
