package middleware

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// newLoggedRouter instruments a router with the access log and captures the standard logger
// output in a hook. The previous level is restored when the test ends.
func newLoggedRouter(t *testing.T, skipPaths []string) (*gin.Engine, *logtest.Hook) {
	t.Helper()
	previous := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(previous) })

	hook := logtest.NewGlobal()
	router := gin.New()
	NewLogger(skipPaths, logrus.InfoLevel).Use(router)
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router, hook
}

func TestLoggerLevelByStatus(t *testing.T) {
	router, hook := newLoggedRouter(t, nil)

	doRequest(router, "/ok")
	doRequest(router, "/missing")

	entries := hook.AllEntries()
	if len(entries) != 2 {
		t.Fatalf("log entries = %d, want 2", len(entries))
	}
	if entries[0].Level != logrus.InfoLevel || entries[1].Level != logrus.ErrorLevel {
		t.Fatalf("levels = %v, %v; want info for 200 and error for 404", entries[0].Level, entries[1].Level)
	}
	if entries[1].Data["status"] != http.StatusNotFound {
		t.Fatalf("status field = %v, want 404", entries[1].Data["status"])
	}
}

func TestLoggerSkipPaths(t *testing.T) {
	router, hook := newLoggedRouter(t, []string{"/healthz"})

	doRequest(router, "/healthz")
	doRequest(router, "/healthz?probe=1")

	if n := len(hook.AllEntries()); n != 0 {
		t.Fatalf("skipped paths produced %d log entries, want 0", n)
	}
}

func TestLoggerSkipPathsMatchesRequestURI(t *testing.T) {
	router, hook := newLoggedRouter(t, []string{"probe=1"})

	doRequest(router, "/ok?probe=1")

	if n := len(hook.AllEntries()); n != 0 {
		t.Fatalf("request matching a skip entry only in the query produced %d entries, want 0", n)
	}
}

func TestLoggerUseAppliesConfiguredLevel(t *testing.T) {
	previous := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(previous) })
	hook := logtest.NewGlobal()
	router := gin.New()

	NewLogger(nil, logrus.ErrorLevel).Use(router)
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	doRequest(router, "/ok")

	if got := logrus.GetLevel(); got != logrus.ErrorLevel {
		t.Fatalf("level after Use = %v, want error", got)
	}
	if n := len(hook.AllEntries()); n != 0 {
		t.Fatalf("200 response at error level produced %d entries, want 0", n)
	}
}

func TestLoggerSetsLevelOnceAtRegistration(t *testing.T) {
	router, _ := newLoggedRouter(t, nil)

	// A level chosen by the application after registration must survive requests.
	logrus.SetLevel(logrus.WarnLevel)
	doRequest(router, "/ok")

	if got := logrus.GetLevel(); got != logrus.WarnLevel {
		t.Fatalf("level after request = %v, want warning (requests must not reset it)", got)
	}
}
