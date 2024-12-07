package azstocker_test

import (
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/calvinmclean/azstocker"
	"github.com/calvinmclean/azstocker/internal/transport"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/sheets/v4"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

const (
	cfpFixture    = "cfp_schedule"
	winterFixture = "winter_schedule"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		program  azstocker.Program
		waters   []string
		expected azstocker.StockingData
	}{
		{
			"Winter_SaltRiver",
			winterFixture,
			azstocker.WinterProgram,
			[]string{"LOWER SALT RIVER"},
			[]azstocker.Calendar{
				{
					WaterName: "LOWER SALT RIVER",
					Data: []azstocker.Week{
						// October
						{
							Year:  2024,
							Month: time.October,
							Day:   1,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   7,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   14,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   21,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   28,
							Stock: azstocker.Trout,
						},
						// November
						{
							Year:  2024,
							Month: time.November,
							Day:   4,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.November,
							Day:   11,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.November,
							Day:   18,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.November,
							Day:   25,
							Stock: azstocker.Trout,
						},
						// December
						{
							Year:  2024,
							Month: time.December,
							Day:   2,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   9,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   16,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   23,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   30,
							Stock: azstocker.NoneFish,
						},
						// January
						{
							Year:  2025,
							Month: time.January,
							Day:   6,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.January,
							Day:   13,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.January,
							Day:   20,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.January,
							Day:   27,
							Stock: azstocker.Trout,
						},
						// February
						{
							Year:  2025,
							Month: time.February,
							Day:   3,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.February,
							Day:   10,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.February,
							Day:   17,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.February,
							Day:   24,
							Stock: azstocker.Trout,
						},
						// March
						{
							Year:  2025,
							Month: time.March,
							Day:   3,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.March,
							Day:   10,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.March,
							Day:   17,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.March,
							Day:   24,
							Stock: azstocker.Trout,
						},
						{
							Year:  2025,
							Month: time.March,
							Day:   31,
							Stock: azstocker.NoneFish,
						},
					},
				},
			},
		},
		{
			"CFP_ManselCarter",
			cfpFixture,
			azstocker.CFProgram,
			[]string{"Queen Creek - Mansel Carter Oasis Lake"},
			[]azstocker.Calendar{
				{
					WaterName: "Queen Creek - Mansel Carter Oasis Lake",
					Data: []azstocker.Week{
						// October
						{
							Year:  2024,
							Month: time.October,
							Day:   7,
							Stock: azstocker.Catfish,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   14,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   21,
							Stock: azstocker.Catfish,
						},
						{
							Year:  2024,
							Month: time.October,
							Day:   28,
							Stock: azstocker.NoneFish,
						},
						// November
						{
							Year:  2024,
							Month: time.November,
							Day:   4,
							Stock: azstocker.Catfish,
						},
						{
							Year:  2024,
							Month: time.November,
							Day:   11,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.November,
							Day:   18,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.November,
							Day:   25,
							Stock: azstocker.NoneFish,
						},
						// December
						{
							Year:  2024,
							Month: time.December,
							Day:   2,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   9,
							Stock: azstocker.Trout,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   16,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   23,
							Stock: azstocker.NoneFish,
						},
						{
							Year:  2024,
							Month: time.December,
							Day:   30,
							Stock: azstocker.Trout,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, r := createTestService(t, tt.fixture)
			defer func() {
				assert.NoError(t, r.Stop())
			}()

			stockData, err := azstocker.Get(srv, tt.program, tt.waters)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, stockData)
		})
	}
}

func TestGetHTTPCache(t *testing.T) {
	numRequests := 0

	srv, r := createTestService(t, winterFixture)
	recorder.WithHook(func(i *cassette.Interaction) error {
		// Set date to now so it is considered "fresh" by the cache
		i.Response.Headers.Set("Date", time.Now().Format(time.RFC1123))

		numRequests++
		return nil
	}, recorder.BeforeResponseReplayHook)(r)
	defer func() {
		assert.NoError(t, r.Stop())
	}()

	_, err := azstocker.Get(srv, azstocker.WinterProgram, []string{})
	assert.NoError(t, err)
	assert.Equal(t, 2, numRequests)

	_, err = azstocker.Get(srv, azstocker.WinterProgram, []string{})
	assert.NoError(t, err)
	assert.Equal(t, 2, numRequests, "no new requests should be created for the 2nd request")
}

func createTestService(t *testing.T, cassetteName string) (*sheets.Service, *recorder.Recorder) {
	t.Helper()

	r, err := recorder.New(
		"internal/testdata/fixtures/"+cassetteName,
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
	srv, err := azstocker.NewService(apiKey, cacheControl)
	assert.NoError(t, err)

	return srv, r
}
