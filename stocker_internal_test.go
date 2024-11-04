package stocker

import (
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/calvinmclean/stocker/internal/transport"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/sheets/v4"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

const (
	cfpFixture    = "cfp_schedule"
	winterFixture = "winter_schedule"
)

func TestParseMonth(t *testing.T) {
	month := func(m time.Month) *time.Month {
		return &m
	}

	tests := []struct {
		input    string
		expected *time.Month
	}{
		{"OCTOBER", month(time.October)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			out := parseMonth(tt.input)
			if tt.expected == nil && out != nil {
				t.Errorf("expected nil but got %v", *out)
			}
			if tt.expected != nil && out == nil {
				t.Errorf("expected %v but got nil", *tt.expected)
			}
			if *tt.expected != *out {
				t.Errorf("expected %v but got %v", *tt.expected, *out)
			}
		})
	}
}

func TestSortNext(t *testing.T) {
	srv, r := createTestService(t, cfpFixture)
	defer func() {
		assert.NoError(t, r.Stop())
	}()

	stockData, _, err := Get(srv, CFProgram, []string{
		"Tempe - Kiwanis Lake",
		"Tempe - Tempe Town Lake",
		"Payson - Green Valley Lakes",
	})
	assert.NoError(t, err)

	getNow = func() time.Time {
		return time.Date(2024, time.November, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() { getNow = time.Now }()

	stockData.SortNext()
	assert.Equal(t, StockingData{
		Calendar{
			WaterName: "Tempe - Kiwanis Lake",
			Data: []Week{
				{Month: 10, Day: 7, Year: 2024, Stock: "Unknown"},
				{Month: 10, Day: 14, Year: 2024, Stock: "None"},
				{Month: 10, Day: 21, Year: 2024, Stock: "Catfish"},
				{Month: 10, Day: 28, Year: 2024, Stock: "None"},
				{Month: 11, Day: 4, Year: 2024, Stock: "Catfish"},
				{Month: 11, Day: 11, Year: 2024, Stock: "None"},
				{Month: 11, Day: 18, Year: 2024, Stock: "None"},
				{Month: 11, Day: 25, Year: 2024, Stock: "None"},
				{Month: 12, Day: 2, Year: 2024, Stock: "None"},
				{Month: 12, Day: 9, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 16, Year: 2024, Stock: "None"},
				{Month: 12, Day: 23, Year: 2024, Stock: "None"},
				{Month: 12, Day: 30, Year: 2024, Stock: "Trout"},
			},
		},
		Calendar{
			WaterName: "Payson - Green Valley Lakes",
			Data: []Week{
				{Month: 10, Day: 7, Year: 2024, Stock: "None"},
				{Month: 10, Day: 14, Year: 2024, Stock: "Trout"},
				{Month: 10, Day: 21, Year: 2024, Stock: "None"},
				{Month: 10, Day: 28, Year: 2024, Stock: "Trout"},
				{Month: 11, Day: 4, Year: 2024, Stock: "None"},
				{Month: 11, Day: 11, Year: 2024, Stock: "None"},
				{Month: 11, Day: 18, Year: 2024, Stock: "Trout"},
				{Month: 11, Day: 25, Year: 2024, Stock: "None"},
				{Month: 12, Day: 2, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 9, Year: 2024, Stock: "None"},
				{Month: 12, Day: 16, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 23, Year: 2024, Stock: "None"},
				{Month: 12, Day: 30, Year: 2024, Stock: "None"},
			},
		},
		Calendar{
			WaterName: "Tempe - Tempe Town Lake",
			Data: []Week{
				{Month: 10, Day: 7, Year: 2024, Stock: "None"},
				{Month: 10, Day: 14, Year: 2024, Stock: "None"},
				{Month: 10, Day: 21, Year: 2024, Stock: "None"},
				{Month: 10, Day: 28, Year: 2024, Stock: "Catfish"},
				{Month: 11, Day: 4, Year: 2024, Stock: "None"},
				{Month: 11, Day: 11, Year: 2024, Stock: "None"},
				{Month: 11, Day: 18, Year: 2024, Stock: "None"},
				{Month: 11, Day: 25, Year: 2024, Stock: "None"},
				{Month: 12, Day: 2, Year: 2024, Stock: "None"},
				{Month: 12, Day: 9, Year: 2024, Stock: "None"},
				{Month: 12, Day: 16, Year: 2024, Stock: "None"},
				{Month: 12, Day: 23, Year: 2024, Stock: "None"},
				{Month: 12, Day: 30, Year: 2024, Stock: "None"},
			},
		},
	}, stockData)
}

func TestSortLast(t *testing.T) {
	srv, r := createTestService(t, cfpFixture)
	defer func() {
		assert.NoError(t, r.Stop())
	}()

	stockData, _, err := Get(srv, CFProgram, []string{
		"St. Johns - Patterson Ponds",
		"Phoenix - Roadrunner Pond",
		"Buckeye - Sundance Park Lake",
	})
	assert.NoError(t, err)

	getNow = func() time.Time {
		return time.Date(2024, time.November, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() { getNow = time.Now }()

	stockData.SortLast()
	assert.Equal(t, StockingData{
		Calendar{
			WaterName: "Phoenix - Roadrunner Pond",
			Data: []Week{
				{Month: 10, Day: 7, Year: 2024, Stock: "None"},
				{Month: 10, Day: 14, Year: 2024, Stock: "None"},
				{Month: 10, Day: 21, Year: 2024, Stock: "None"},
				{Month: 10, Day: 28, Year: 2024, Stock: "Catfish"},
				{Month: 11, Day: 4, Year: 2024, Stock: "None"},
				{Month: 11, Day: 11, Year: 2024, Stock: "None"},
				{Month: 11, Day: 18, Year: 2024, Stock: "None"},
				{Month: 11, Day: 25, Year: 2024, Stock: "None"},
				{Month: 12, Day: 2, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 9, Year: 2024, Stock: "None"},
				{Month: 12, Day: 16, Year: 2024, Stock: "None"},
				{Month: 12, Day: 23, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 30, Year: 2024, Stock: "None"},
			},
		},
		Calendar{
			WaterName: "Buckeye - Sundance Park Lake",
			Data: []Week{
				{Month: 10, Day: 7, Year: 2024, Stock: "Catfish"},
				{Month: 10, Day: 14, Year: 2024, Stock: "None"},
				{Month: 10, Day: 21, Year: 2024, Stock: "Catfish"},
				{Month: 10, Day: 28, Year: 2024, Stock: "None"},
				{Month: 11, Day: 4, Year: 2024, Stock: "Catfish"},
				{Month: 11, Day: 11, Year: 2024, Stock: "None"},
				{Month: 11, Day: 18, Year: 2024, Stock: "None"},
				{Month: 11, Day: 25, Year: 2024, Stock: "None"},
				{Month: 12, Day: 2, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 9, Year: 2024, Stock: "None"},
				{Month: 12, Day: 16, Year: 2024, Stock: "Trout"},
				{Month: 12, Day: 23, Year: 2024, Stock: "None"},
				{Month: 12, Day: 30, Year: 2024, Stock: "None"},
			},
		},
		Calendar{
			WaterName: "St. Johns - Patterson Ponds",
			Data: []Week{
				{Month: 10, Day: 7, Year: 2024, Stock: "None"},
				{Month: 10, Day: 14, Year: 2024, Stock: "Trout"},
				{Month: 10, Day: 21, Year: 2024, Stock: "None"},
				{Month: 10, Day: 28, Year: 2024, Stock: "None"},
				{Month: 11, Day: 4, Year: 2024, Stock: "None"},
				{Month: 11, Day: 11, Year: 2024, Stock: "None"},
				{Month: 11, Day: 18, Year: 2024, Stock: "Trout"},
				{Month: 11, Day: 25, Year: 2024, Stock: "None"},
				{Month: 12, Day: 2, Year: 2024, Stock: "None"},
				{Month: 12, Day: 9, Year: 2024, Stock: "None"},
				{Month: 12, Day: 16, Year: 2024, Stock: "None"},
				{Month: 12, Day: 23, Year: 2024, Stock: "None"},
				{Month: 12, Day: 30, Year: 2024, Stock: "None"},
			},
		},
	}, stockData)
}

func TestNextLast(t *testing.T) {
	srv, r := createTestService(t, cfpFixture)
	defer func() {
		assert.NoError(t, r.Stop())
	}()

	stockData, _, err := Get(srv, CFProgram, []string{"Queen Creek - Mansel Carter Oasis Lake"})
	assert.NoError(t, err)

	getNow = func() time.Time {
		return time.Date(2024, time.November, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() { getNow = time.Now }()

	t.Run("Last", func(t *testing.T) {
		last := stockData[0].Last()
		assert.Equal(t, Week{
			Month: time.October,
			Day:   21,
			Year:  2024,
			Stock: Catfish,
		}, last)
	})

	t.Run("Last", func(t *testing.T) {
		next := stockData[0].Next()
		assert.Equal(t, Week{
			Month: time.November,
			Day:   4,
			Year:  2024,
			Stock: Catfish,
		}, next)
	})
}

func createTestService(t *testing.T, cassetteName string) (*sheets.Service, *recorder.Recorder) {
	t.Helper()

	r, err := recorder.New(
		"fixtures/"+cassetteName,
		recorder.WithSkipRequestLatency(true),
		// Don't save API Key in fixtures
		recorder.WithHook(func(i *cassette.Interaction) error {
			parsedURL, err := url.Parse(i.Request.URL)
			if err != nil {
				return err
			}

			query := parsedURL.Query()
			query.Del("key")

			parsedURL.RawQuery = query.Encode()
			i.Request.URL = parsedURL.String()
			return nil
		}, recorder.BeforeSaveHook),
		// Ignore API key in matching. All requests are simple GET requests so we just need to match URL
		recorder.WithMatcher(func(r *http.Request, i cassette.Request) bool {
			query := r.URL.Query()
			query.Del("key")

			r.URL.RawQuery = query.Encode()
			return r.Method == i.Method && r.URL.String() == i.URL
		}),
	)
	assert.NoError(t, err)

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = "PLACEHOLDER"
	} else {
		recorder.WithMode(recorder.ModeRecordOnly)(r)
	}

	cacheControl := transport.NewCacheControl(time.Minute, r)
	srv, err := NewService(apiKey, cacheControl)
	assert.NoError(t, err)

	return srv, r
}
