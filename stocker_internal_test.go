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

	stockData, _, err := Get(srv, CFProgram, []string{})
	assert.NoError(t, err)

	getNow = func() time.Time {
		return time.Date(2024, time.November, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() { getNow = time.Now }()

	nextWaters := SortNext(stockData)
	assert.Equal(t, []map[string]Week{
		{"Buckeye - Sundance Park Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Chandler - Desert Breeze Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Chandler - Veterans Oasis Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Gila Bend - Benders Pond (NEW)": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Maricopa - Copper Sky Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Mesa - Red Mountain Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Mesa - Riverview Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Peoria - Paloma Park": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Peoria - Pioneer Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Peoria - Rio Vista Pond": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Alvord Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Cortez Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Desert West Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Encanto Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Papago Ponds": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Steele Indian School Pond": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Queen Creek - Mansel Carter Oasis Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Sahuarita - Sahuarita Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Scottsdale - Chaparral Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Somerton - Council Avenue Pond": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Surprise - Surprise Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Tempe - Kiwanis Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Tucson - Kennedy Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Tucson - Lakeside Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Tucson - Silverbell Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Yuma - Fortuna Lake": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Yuma - PAAC Pond": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Yuma - West Wetlands Pond": {Month: 11, Day: 4, Year: 2024, Stock: "Catfish"}},
		{"Payson - Green Valley Lakes": {Month: 11, Day: 18, Year: 2024, Stock: "Trout"}},
		{"Prescott Valley - Fain Lake": {Month: 11, Day: 18, Year: 2024, Stock: "Trout"}},
		{"Prescott Valley - Yavapai Lakes (Urban Forest Park)": {Month: 11, Day: 18, Year: 2024, Stock: "Trout"}},
		{"Show Low Creek (Meadow at Bluff Trail)": {Month: 11, Day: 18, Year: 2024, Stock: "Trout"}},
		{"St. Johns - Patterson Ponds": {Month: 11, Day: 18, Year: 2024, Stock: "Trout"}},
		{"Avondale - Alamar Park Pond": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Avondale - Festival Fields Pond": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Avondale - Friendship Pond": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Glendale - Bonsall Pond": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Glendale - Heroes Regional Park Pond": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Phoenix - Roadrunner Pond": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Safford - Graham County Fairgrounds": {Month: 12, Day: 2, Year: 2024, Stock: "Trout"}},
		{"Casa Grande - Dave White Regional Park": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Gilbert - Discovery Ponds": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Gilbert - Freestone Pond": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Gilbert - Gilbert Regional Park": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Gilbert - McQueen Pond": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Gilbert - Water Ranch Lake *(Special Regulations)": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Maricopa - Pacana Pond": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Mesa - Eastmark Phase 4 Pond": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
		{"Mesa - Greenfield Pond": {Month: 12, Day: 9, Year: 2024, Stock: "Trout"}},
	}, nextWaters)
}

func TestSortLast(t *testing.T) {
	srv, r := createTestService(t, cfpFixture)
	defer func() {
		assert.NoError(t, r.Stop())
	}()

	stockData, _, err := Get(srv, CFProgram, []string{})
	assert.NoError(t, err)

	getNow = func() time.Time {
		return time.Date(2024, time.November, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() { getNow = time.Now }()

	nextWaters := SortLast(stockData)
	assert.Equal(t, []map[string]Week{
		{"Avondale - Alamar Park Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Unknown"}},
		{"Avondale - Festival Fields Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Unknown"}},
		{"Avondale - Friendship Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Casa Grande - Dave White Regional Park": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Gilbert - Discovery Ponds": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Gilbert - Freestone Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Gilbert - Gilbert Regional Park": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Gilbert - McQueen Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Gilbert - Water Ranch Lake *(Special Regulations)": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Glendale - Bonsall Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Glendale - Heroes Regional Park Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Maricopa - Pacana Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Mesa - Eastmark Phase 4 Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Mesa - Greenfield Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Payson - Green Valley Lakes": {Month: 10, Day: 28, Year: 2024, Stock: "Trout"}},
		{"Phoenix - Roadrunner Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Show Low Creek (Meadow at Bluff Trail)": {Month: 10, Day: 28, Year: 2024, Stock: "Trout"}},
		{"Tempe - Evelyn Hallman Pond": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Tempe - Tempe Town Lake": {Month: 10, Day: 28, Year: 2024, Stock: "Catfish"}},
		{"Buckeye - Sundance Park Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Chandler - Desert Breeze Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Unknown"}},
		{"Chandler - Veterans Oasis Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Maricopa - Copper Sky Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Unknown"}},
		{"Mesa - Red Mountain Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Mesa - Riverview Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Peoria - Paloma Park": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Peoria - Pioneer Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Peoria - Rio Vista Pond": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Alvord Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Cortez Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Desert West Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Encanto Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Papago Ponds": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Phoenix - Steele Indian School Pond": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Queen Creek - Mansel Carter Oasis Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Safford - Graham County Fairgrounds": {Month: 10, Day: 21, Year: 2024, Stock: "Unknown"}},
		{"Sahuarita - Sahuarita Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Scottsdale - Chaparral Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Surprise - Surprise Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Tempe - Kiwanis Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Tucson - Kennedy Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Tucson - Lakeside Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Tucson - Silverbell Lake": {Month: 10, Day: 21, Year: 2024, Stock: "Catfish"}},
		{"Prescott Valley - Fain Lake": {Month: 10, Day: 14, Year: 2024, Stock: "Trout"}},
		{"Prescott Valley - Yavapai Lakes (Urban Forest Park)": {Month: 10, Day: 14, Year: 2024, Stock: "Trout"}},
		{"St. Johns - Patterson Ponds": {Month: 10, Day: 14, Year: 2024, Stock: "Trout"}},
		{"Gila Bend - Benders Pond (NEW)": {Month: 10, Day: 7, Year: 2024, Stock: "Catfish"}},
		{"Somerton - Council Avenue Pond": {Month: 10, Day: 7, Year: 2024, Stock: "Catfish"}},
		{"Yuma - Fortuna Lake": {Month: 10, Day: 7, Year: 2024, Stock: "Catfish"}},
		{"Yuma - PAAC Pond": {Month: 10, Day: 7, Year: 2024, Stock: "Catfish"}},
		{"Yuma - West Wetlands Pond": {Month: 10, Day: 7, Year: 2024, Stock: "Catfish"}},
	}, nextWaters)
}

func TestNextLast(t *testing.T) {
	srv, r := createTestService(t, cfpFixture)
	defer func() {
		assert.NoError(t, r.Stop())
	}()

	water := "Queen Creek - Mansel Carter Oasis Lake"
	stockData, _, err := Get(srv, CFProgram, []string{water})
	assert.NoError(t, err)

	getNow = func() time.Time {
		return time.Date(2024, time.November, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() { getNow = time.Now }()

	t.Run("Last", func(t *testing.T) {
		last := stockData[water].Last()
		assert.Equal(t, Week{
			Month: time.October,
			Day:   21,
			Year:  2024,
			Stock: Catfish,
		}, last)
	})

	t.Run("Last", func(t *testing.T) {
		next := stockData[water].Next()
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
