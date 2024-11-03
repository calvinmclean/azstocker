package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/calvinmclean/stocker"

	"google.golang.org/api/sheets/v4"
)

const watersQueryParam = "waters"

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

	data, err := stocker.Get(s.srv, program, waters)
	if err != nil {
		fmt.Fprintf(w, "error: %v", err)
		return
	}

	for waterName, calendar := range data {
		fmt.Fprintln(w, waterName)
		fmt.Fprintln(w, calendar.DetailFormat(showAll, showAllStock, next, last))
	}
}

type query struct {
	r *http.Request
}

func (q query) Bool(key string) bool {
	return strings.ToLower(q.r.URL.Query().Get(key)) == "true"
}
