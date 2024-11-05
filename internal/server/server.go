package server

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/calvinmclean/stocker"

	"google.golang.org/api/sheets/v4"
)

const (
	watersQueryParam = "waters"
	templateFilename = "templates/*"
)

//go:embed templates/*
var templateFS embed.FS

func RunServer(addr string, srv *sheets.Service) error {
	mux := http.NewServeMux()

	s := &server{srv}
	mux.HandleFunc("/", s.homepage)
	mux.HandleFunc("/{program}", s.getProgramSchedule)

	return http.ListenAndServe(addr, mux)
}

type server struct {
	srv *sheets.Service
}

func (s *server) homepage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := loadTemplates()
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to parse template", "err", err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(w, "homepage", nil)
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to execute template", "err", err.Error())
		return
	}
}

func (s *server) getProgramSchedule(w http.ResponseWriter, r *http.Request) {
	var waters []string
	if r.URL.Query().Has(watersQueryParam) {
		waters = strings.Split(r.URL.Query().Get(watersQueryParam), ",")
	}

	programStr := r.PathValue("program")
	program, err := stocker.ParseProgram(programStr)
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "invalid program", "program", programStr, "err", err.Error())
		return
	}

	q := query{r}
	showAll := q.Bool("showAll")
	sortBy := r.URL.Query().Get("sortBy")

	stockingData, err := stocker.Get(s.srv, program, waters)
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
		stockingData.Sort(func(c1, c2 stocker.Calendar) int { return 0 })
	}

	tmpl, err := loadTemplates()
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to parse template", "err", err.Error())
		return
	}

	watersStr := strings.Join(waters, ", ")
	err = tmpl.ExecuteTemplate(w, "calendar", map[string]any{
		"showAll":  showAll,
		"program":  program,
		"calendar": stockingData,
		"waters":   watersStr,
		"sortedBy": sortBy,
	})
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to execute template", "err", err.Error())
		return
	}
}

type query struct {
	r *http.Request
}

func (q query) Bool(key string) bool {
	return strings.ToLower(q.r.URL.Query().Get(key)) == "true"
}

func loadTemplates() (*template.Template, error) {
	if os.Getenv("DEV") == "true" {
		_, callerFile, _, _ := runtime.Caller(0)
		return template.ParseGlob(filepath.Join(filepath.Dir(callerFile), templateFilename))
	}

	return template.ParseFS(templateFS, templateFilename)
}
