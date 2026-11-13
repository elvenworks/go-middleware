package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// captureStandardLogger attaches a test hook to the standard logrus logger as its only hook and
// returns it. The previous level and hooks are restored when the test ends.
func captureStandardLogger(t *testing.T) *logtest.Hook {
	t.Helper()
	previousLevel := logrus.GetLevel()
	previousHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(previousHooks)
		logrus.SetLevel(previousLevel)
	})
	return logtest.NewGlobal()
}

// newLoggedRouter instruments a router with the access log and captures the standard logger
// output in a hook. The previous level and hooks are restored when the test ends.
func newLoggedRouter(t *testing.T, skipPaths []string) (*gin.Engine, *logtest.Hook) {
	t.Helper()
	hook := captureStandardLogger(t)
	router := gin.New()
	NewLogger(skipPaths, logrus.InfoLevel).Use(router)
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/bad", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
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
	hook := captureStandardLogger(t)
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

func TestLoggerRedactsSensitiveQuery(t *testing.T) {
	router, hook := newLoggedRouter(t, nil)

	doRequest(router, "/ok?token=s3cr3t-fake&orgUid=o1")

	entries := hook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	if got := entries[0].Data["path"]; got != "/ok?token=REDACTED&orgUid=o1" {
		t.Fatalf("path field = %v, want redacted query", got)
	}
	for _, e := range entries {
		if strings.Contains(fmt.Sprint(e.Data), "s3cr3t-fake") || strings.Contains(e.Message, "s3cr3t-fake") {
			t.Fatalf("log entry leaks the secret: %v %q", e.Data, e.Message)
		}
	}
}

func TestLoggerRedactsOnErrorStatus(t *testing.T) {
	router, hook := newLoggedRouter(t, nil)

	doRequest(router, "/bad?token=s3cr3t-fake")

	entries := hook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	if entries[0].Level != logrus.ErrorLevel {
		t.Fatalf("level = %v, want error", entries[0].Level)
	}
	if got := entries[0].Data["path"]; got != "/bad?token=REDACTED" {
		t.Fatalf("path field = %v, want redacted query", got)
	}
}

func TestLoggerLogsRequestAsReceived(t *testing.T) {
	router, hook := newLoggedRouter(t, nil)
	router.GET("/rewrite", func(c *gin.Context) {
		c.Request.URL.Path = "/rewritten"
		c.Request.URL.RawQuery = "token=other-fake"
		c.Status(http.StatusOK)
	})

	doRequest(router, "/rewrite?token=s3cr3t-fake&orgUid=o1")

	entries := hook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	if got := entries[0].Data["path"]; got != "/rewrite?token=REDACTED&orgUid=o1" {
		t.Fatalf("path field = %v, want the received URI, redacted", got)
	}
}

func TestLoggerSkipUsesRequestAsReceived(t *testing.T) {
	router, hook := newLoggedRouter(t, []string{"/healthz"})
	router.GET("/healthz/rewrite", func(c *gin.Context) {
		c.Request.URL.Path = "/elsewhere"
		c.Status(http.StatusOK)
	})

	doRequest(router, "/healthz/rewrite")

	if n := len(hook.AllEntries()); n != 0 {
		t.Fatalf("skipped path rewritten by the handler produced %d entries, want 0", n)
	}
}

func TestLoggerKeepsQueryWithoutSecrets(t *testing.T) {
	router, hook := newLoggedRouter(t, nil)

	doRequest(router, "/ok?page=2&size=10")

	entries := hook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	if got := entries[0].Data["path"]; got != "/ok?page=2&size=10" {
		t.Fatalf("path field = %v, want unchanged query", got)
	}
}
