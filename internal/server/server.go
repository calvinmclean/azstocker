package server

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/calvinmclean/stocker"

	"google.golang.org/api/sheets/v4"
)

const (
	watersQueryParam = "waters"
	templateFilename = "templates/calendar.html.tmpl"
)

//go:embed templates/*
var templateFS embed.FS

func RunServer(addr string, srv *sheets.Service) error {
	mux := http.NewServeMux()

	s := &server{srv}
	mux.HandleFunc("/{program}", s.getProgramSchedule)

	return http.ListenAndServe(addr, mux)
}

type server struct {
	srv *sheets.Service
}

func (s *server) getProgramSchedule(w http.ResponseWriter, r *http.Request) {
	var waters []string
	if r.URL.Query().Has(watersQueryParam) {
		waters = strings.Split(r.URL.Query().Get(watersQueryParam), ",")
	}
	program := r.PathValue("program")

	q := query{r}
	showAll := q.Bool("showAll")
	showAllStock := q.Bool("showAllStock")
	next := q.Bool("next")
	last := q.Bool("last")

	calendar, allWaterNames, err := stocker.Get(s.srv, program, waters)
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to get data", "err", err.Error())
		return
	}

	slices.Sort(allWaterNames)

	var tmpl *template.Template
	if os.Getenv("DEV") == "true" {
		_, callerFile, _, _ := runtime.Caller(0)
		tmpl, err = template.ParseFiles(filepath.Join(filepath.Dir(callerFile), templateFilename))
	} else {
		tmpl, err = template.ParseFS(templateFS, templateFilename)
	}
	if err != nil {
		slog.Log(r.Context(), slog.LevelError, "failed to parse template", "err", err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(w, "calendar", map[string]any{
		"showAll":       showAll,
		"showAllStock":  showAllStock,
		"next":          next,
		"last":          last,
		"program":       program,
		"calendar":      calendar,
		"allWaterNames": allWaterNames,
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
