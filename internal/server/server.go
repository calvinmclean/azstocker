package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/calvinmclean/azstocker"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	prommetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metrics_middleware "github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
	"google.golang.org/api/sheets/v4"
)

const (
	watersQueryParam = "waters"
	templateFilename = "templates/*"

	metricsAddr = "0.0.0.0:9091"
)

var (
	programsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "azstocker",
		Name:      "program_requests",
		Help:      "gauge of programs requested",
	}, []string{"program"})

	watersGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "azstocker",
		Name:      "water_requests",
		Help:      "gauge of waters requested",
	}, []string{"water"})
)

func init() {
	prometheus.MustRegister(programsGauge, watersGauge)
}

//go:embed templates/*
var templateFS embed.FS

type Option func(*server) error

func WithPushoverClient(appToken, recipientToken string) Option {
	return func(s *server) error {
		nc, err := newNotifyClient(appToken, recipientToken)
		if err != nil {
			return fmt.Errorf("error initializing notify client: %w", err)
		}
		s.nc = nc
		s.notifySourceIPs = &sync.Map{}
		return nil
	}
}

func RunServer(addr string, srv *sheets.Service, urlBase string, opts ...Option) error {
	mux, err := newServer(srv, urlBase, opts...)
	if err != nil {
		return err
	}

	metricServer, metricMiddleware := newMetricsServer()
	go func() {
		metricErr := http.ListenAndServe(metricsAddr, metricServer)
		if err != nil {
			slog.Log(context.Background(), slog.LevelError, "error running metrics server", "err", metricErr.Error())
		}
	}()
	return http.ListenAndServe(addr, metricMiddleware(mux))
}

func newMetricsServer() (*http.ServeMux, func(http.Handler) http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	middleware := std.HandlerProvider("", metrics_middleware.New(metrics_middleware.Config{
		Recorder: prommetrics.NewRecorder(prommetrics.Config{Prefix: "azstocker"}),
	}))

	return mux, middleware
}

func newServer(srv *sheets.Service, urlBase string, opts ...Option) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	s := &server{srv, urlBase, nil, nil}
	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}
	mux.HandleFunc("/", s.homepage)
	mux.HandleFunc("/index.html", s.homepage)
	mux.HandleFunc("/sitemap.txt", s.sitemap)
	if s.nc != nil {
		mux.HandleFunc("/notify", s.notify)
	}
	mux.HandleFunc("/manifest.json", s.pwaManifest)
	mux.HandleFunc("/{program}", s.getProgramSchedule)

	return mux, nil
}

type server struct {
	srv     *sheets.Service
	urlBase string

	nc              *notifyClient
	notifySourceIPs *sync.Map
}

// manifest.json enables PWA for mobile devices
func (s *server) pwaManifest(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{
	  "name": "AZ Stocker",
	  "start_url": "/",
	  "display": "standalone"
	}`))
}

func (s *server) homepage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := loadTemplates()
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to parse template", "err", err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(w, "homepage", map[string]any{
		"notifyEnabled": s.notifyEnabled(r),
		"program":       "home",
	})
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to execute template", "err", err.Error())
		return
	}
}

func (s *server) sitemap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.writeSitemap(r.Context(), w)
}

func (s *server) writeSitemap(ctx context.Context, w io.Writer) {
	programs := []azstocker.Program{azstocker.CFProgram, azstocker.WinterProgram, azstocker.SpringSummerProgram}
	for _, p := range programs {
		stockingData, err := azstocker.Get(s.srv, p, []string{})
		if err != nil {
			slog.Log(ctx, slog.LevelError, "failed to get data", "err", err.Error())
		}

		stockingData.Sort(func(c1, c2 azstocker.Calendar) int {
			return strings.Compare(c1.WaterName, c2.WaterName)
		})

		urlBase := fmt.Sprintf("%s/%s", s.urlBase, p)
		fmt.Fprintf(w, "%s\n", urlBase)

		for _, data := range stockingData {
			query := url.Values{
				"waters": []string{data.WaterName},
			}
			fmt.Fprintf(w, "%s?%s\n", urlBase, query.Encode())
		}
	}
}

func (s *server) getProgramSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	programStr := r.PathValue("program")
	program, err := azstocker.ParseProgram(programStr)
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "invalid program", "program", programStr, "err", err.Error())
		return
	}

	programsGauge.WithLabelValues(programStr).Inc()

	q := query{r}
	showAll := q.Bool("showAll")
	sortBy := r.URL.Query().Get("sortBy")
	waters := q.StringSlice("waters")
	if len(waters) > 0 {
		for _, w := range waters {
			watersGauge.WithLabelValues(w).Inc()
		}
	}

	stockingData, err := azstocker.Get(s.srv, program, waters)
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to get data", "err", err.Error())
		return
	}

	switch sortBy {
	case "next":
		stockingData.SortNext()
	case "last":
		stockingData.SortLast()
	case "":
		stockingData.Sort(func(c1, c2 azstocker.Calendar) int { return 0 })
	}

	tmpl, err := loadTemplates()
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to parse template", "err", err.Error())
		return
	}

	watersStr := strings.Join(waters, ", ")
	err = tmpl.ExecuteTemplate(w, "calendar", map[string]any{
		"showAll":       showAll,
		"program":       program,
		"calendar":      stockingData,
		"waters":        watersStr,
		"numWaters":     len(waters),
		"sortedBy":      sortBy,
		"notifyEnabled": s.notifyEnabled(r),
	})
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to execute template", "err", err.Error())
		return
	}
}

func (s *server) notify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	remoteAddr := remoteAddr(r)

	_, exists := s.notifySourceIPs.LoadOrStore(remoteAddr, struct{}{})
	if exists {
		slog.Log(r.Context(), slog.LevelWarn, "notify from repeated IP", "remote_addr", remoteAddr)
		return
	}

	slog.Log(r.Context(), slog.LevelInfo, "received notify request", "remote_addr", remoteAddr)

	err := s.nc.send("AZStocker", "AZStocker got a like!")
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "error notifying", "err", err)
		return
	}
}

func (s *server) notifyEnabled(r *http.Request) bool {
	if s.nc == nil {
		return false
	}

	_, alreadyNotified := s.notifySourceIPs.Load(remoteAddr(r))
	return !alreadyNotified
}

func remoteAddr(r *http.Request) string {
	remoteAddr := r.RemoteAddr
	i := strings.LastIndex(r.RemoteAddr, ":")
	if i > 0 {
		remoteAddr = remoteAddr[0:i]
	}
	return remoteAddr
}

type query struct {
	r *http.Request
}

func (q query) Bool(key string) bool {
	return strings.ToLower(q.r.URL.Query().Get(key)) == "true"
}

func (q query) StringSlice(key string) []string {
	result := []string{}
	if !q.r.URL.Query().Has(watersQueryParam) {
		return result
	}

	rawQuerySlice := strings.Split(q.r.URL.Query().Get(watersQueryParam), ",")
	for _, w := range rawQuerySlice {
		result = append(result, strings.TrimSpace(w))
	}
	return result
}

func loadTemplates() (*template.Template, error) {
	tmpl := template.New("template").Funcs(template.FuncMap{
		"escapeSingleQuote": func(in string) string {
			return strings.ReplaceAll(in, "'", "\\'")
		},
	})

	if os.Getenv("DEV") == "true" {
		_, callerFile, _, _ := runtime.Caller(0)
		return tmpl.ParseGlob(filepath.Join(filepath.Dir(callerFile), templateFilename))
	}
	return tmpl.ParseFS(templateFS, templateFilename)
}
