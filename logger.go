package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type LoggerMiddleware struct {
	SkipPaths []string
	Level     logrus.Level
}

// Use registers the access log middleware on r and applies Level to the standard logrus logger
// once, at registration, instead of on every request.
//
// Example: NewLogger([]string{"/metrics"}, logrus.InfoLevel).Use(router)
func (m LoggerMiddleware) Use(r *gin.Engine) {
	logrus.SetLevel(m.Level)

	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		url := c.Request.URL.RequestURI()

		c.Next()

		if m.shouldSkip(path, url) {
			return
		}
		m.logRequest(c, url, time.Since(start))
	})
}

// shouldSkip checks both the decoded path and the raw request URI, so a skip entry matches
// whether or not the request path was percent-encoded.
func (m LoggerMiddleware) shouldSkip(path, url string) bool {
	for _, p := range m.SkipPaths {
		if strings.Contains(path, p) || strings.Contains(url, p) {
			return true
		}
	}
	return false
}

func (m LoggerMiddleware) logRequest(c *gin.Context, url string, latency time.Duration) {
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

	if c.Writer.Status() > 299 {
		log.Error(msg)
	} else {
		log.Info(msg)
	}
}

// NewLogger creates an access log middleware that skips any request whose path contains one of
// skipPaths and logs responses with status above 299 as errors.
//
// Example: NewLogger([]string{"/healthz", "/metrics"}, logrus.InfoLevel).Use(router)
func NewLogger(skipPaths []string, level logrus.Level) *LoggerMiddleware {
	return &LoggerMiddleware{
		SkipPaths: skipPaths,
		Level:     level,
	}
}
