package middleware

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter builds a router instrumented by p with one parameterized route. Each test uses
// its own subsystem because the middleware registers into the global Prometheus registry.
func newTestRouter(p *Prometheus) *gin.Engine {
	router := gin.New()
	p.Use(router)
	router.GET("/alerts/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func doRequest(router http.Handler, path string) int {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder.Code
}

// urlLabels returns the sorted distinct "url" label values recorded by a counter vector.
func urlLabels(t *testing.T, vec *prometheus.CounterVec) []string {
	t.Helper()
	ch := make(chan prometheus.Metric, 100)
	vec.Collect(ch)
	close(ch)

	seen := map[string]bool{}
	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err != nil {
			t.Fatalf("writing metric: %v", err)
		}
		for _, label := range m.GetLabel() {
			if label.GetName() == "url" {
				seen[label.GetValue()] = true
			}
		}
	}
	labels := make([]string, 0, len(seen))
	for label := range seen {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}

func assertLabels(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("url labels = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("url labels = %v, want %v", got, want)
		}
	}
}

func TestDefaultURLLabelUsesRouteTemplate(t *testing.T) {
	p := NewPrometheus("test_route_template")
	router := newTestRouter(p)

	for _, path := range []string{"/alerts/1", "/alerts/2", "/alerts/3", "/does-not-exist", "/scan/../etc"} {
		doRequest(router, path)
	}

	assertLabels(t, urlLabels(t, p.reqCnt), "/alerts/:id", UnmatchedRouteLabel)
}

func TestMethodNotAllowedIsUnmatched(t *testing.T) {
	p := NewPrometheus("test_method_not_allowed")
	router := newTestRouter(p)
	router.HandleMethodNotAllowed = true

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/alerts/1", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /alerts/1 status = %d, want 405", recorder.Code)
	}
	assertLabels(t, urlLabels(t, p.reqCnt), UnmatchedRouteLabel)
}

func TestUnrelatedCollectorWithSameNameDoesNotPanic(t *testing.T) {
	// Same name and help as the request size summary, but a different collector type.
	conflicting := prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: "test_conflict", Name: reqSz.Name, Help: reqSz.Description,
	})
	if err := prometheus.Register(conflicting); err != nil {
		t.Fatalf("registering conflicting collector: %v", err)
	}
	t.Cleanup(func() { prometheus.Unregister(conflicting) })

	p := NewPrometheus("test_conflict")
	doRequest(newTestRouter(p), "/alerts/1")

	assertLabels(t, urlLabels(t, p.reqCnt), "/alerts/:id")
}

func TestPushMetricsSkipsNon2xxMetricsResponse(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("push gateway must not receive an error page as metrics")
	}))
	defer gateway.Close()
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failing.Close()

	p := &Prometheus{Ppg: PrometheusPushGateway{PushGatewayURL: gateway.URL, MetricsURL: failing.URL}}
	p.pushMetrics()
}

func TestRawPathURLLabelKeepsLegacyBehavior(t *testing.T) {
	p := NewPrometheus("test_raw_path")
	p.ReqCntURLLabelMappingFn = RawPathURLLabel
	router := newTestRouter(p)

	doRequest(router, "/alerts/1")
	doRequest(router, "/alerts/2")

	assertLabels(t, urlLabels(t, p.reqCnt), "/alerts/1", "/alerts/2")
}

func TestMetricsPathIsNotCounted(t *testing.T) {
	p := NewPrometheus("test_metrics_path")
	router := newTestRouter(p)

	if code := doRequest(router, "/metrics"); code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want 200", code)
	}

	assertLabels(t, urlLabels(t, p.reqCnt))
}

func TestURLLabelFromContext(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "string value", value: "/custom", want: "/custom"},
		{name: "non-string value", value: 42, want: "unknown"},
		{name: "missing value", value: nil, want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &Prometheus{URLLabelFromContext: "route", ReqCntURLLabelMappingFn: RouteTemplateURLLabel}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tc.value != nil {
				c.Set("route", tc.value)
			}
			if got := p.urlLabel(c); got != tc.want {
				t.Fatalf("urlLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSecondInstanceSharesRegisteredCollectors(t *testing.T) {
	first := NewPrometheus("test_shared")
	second := NewPrometheus("test_shared")

	doRequest(newTestRouter(second), "/alerts/1")

	if first.reqCnt != second.reqCnt {
		t.Fatal("second NewPrometheus with the same subsystem must reuse the registered collector")
	}
	assertLabels(t, urlLabels(t, first.reqCnt), "/alerts/:id")
}

func TestPushMetricsSkipsUnreachableEndpoint(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("push gateway must not be called when metrics cannot be read")
	}))
	defer gateway.Close()

	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	p := &Prometheus{Ppg: PrometheusPushGateway{PushGatewayURL: gateway.URL, MetricsURL: closed.URL}}
	p.pushMetrics()
}
