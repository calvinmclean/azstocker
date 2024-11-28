package server

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/calvinmclean/stocker"
	"github.com/calvinmclean/stocker/internal/transport"
	"github.com/stretchr/testify/assert"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func TestWriteSitemap(t *testing.T) {
	r, err := recorder.New(
		"../testdata/fixtures/both",
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
	srv, err := stocker.NewService(apiKey, cacheControl)
	assert.NoError(t, err)

	expected := `http://example.com/cfp
http://example.com/cfp?waters=Avondale+-+Alamar+Park+Pond
http://example.com/cfp?waters=Avondale+-+Festival+Fields+Pond
http://example.com/cfp?waters=Avondale+-+Friendship+Pond
http://example.com/cfp?waters=Buckeye+-+Sundance+Park+Lake
http://example.com/cfp?waters=Casa+Grande+-+Dave+White+Regional+Park
http://example.com/cfp?waters=Chandler+-+Desert+Breeze+Lake
http://example.com/cfp?waters=Chandler+-+Veterans+Oasis+Lake
http://example.com/cfp?waters=Gila+Bend+-+Benders+Pond+%28NEW%29
http://example.com/cfp?waters=Gilbert+-+Discovery+Ponds
http://example.com/cfp?waters=Gilbert+-+Freestone+Pond
http://example.com/cfp?waters=Gilbert+-+Gilbert+Regional+Park
http://example.com/cfp?waters=Gilbert+-+McQueen+Pond
http://example.com/cfp?waters=Gilbert+-+Water+Ranch+Lake+%2A%28Special+Regulations%29
http://example.com/cfp?waters=Glendale+-+Bonsall+Pond
http://example.com/cfp?waters=Glendale+-+Heroes+Regional+Park+Pond
http://example.com/cfp?waters=Maricopa+-+Copper+Sky+Lake
http://example.com/cfp?waters=Maricopa+-+Pacana+Pond
http://example.com/cfp?waters=Mesa+-+Eastmark+Phase+4+Pond
http://example.com/cfp?waters=Mesa+-+Greenfield+Pond
http://example.com/cfp?waters=Mesa+-+Red+Mountain+Lake
http://example.com/cfp?waters=Mesa+-+Riverview+Lake
http://example.com/cfp?waters=Payson+-+Green+Valley+Lakes
http://example.com/cfp?waters=Peoria+-+Paloma+Park
http://example.com/cfp?waters=Peoria+-+Pioneer+Lake
http://example.com/cfp?waters=Peoria+-+Rio+Vista+Pond
http://example.com/cfp?waters=Phoenix+-+Alvord+Lake
http://example.com/cfp?waters=Phoenix+-+Cortez+Lake
http://example.com/cfp?waters=Phoenix+-+Desert+West+Lake
http://example.com/cfp?waters=Phoenix+-+Encanto+Lake
http://example.com/cfp?waters=Phoenix+-+Papago+Ponds
http://example.com/cfp?waters=Phoenix+-+Roadrunner+Pond
http://example.com/cfp?waters=Phoenix+-+Steele+Indian+School+Pond
http://example.com/cfp?waters=Prescott+Valley+-+Fain+Lake
http://example.com/cfp?waters=Prescott+Valley+-+Yavapai+Lakes+%28Urban+Forest+Park%29
http://example.com/cfp?waters=Queen+Creek+-+Mansel+Carter+Oasis+Lake
http://example.com/cfp?waters=Safford+-+Graham+County+Fairgrounds
http://example.com/cfp?waters=Sahuarita+-+Sahuarita+Lake
http://example.com/cfp?waters=Scottsdale+-+Chaparral+Lake
http://example.com/cfp?waters=Show+Low+Creek+%28Meadow+at+Bluff+Trail%29
http://example.com/cfp?waters=Somerton+-+Council+Avenue+Pond
http://example.com/cfp?waters=St.+Johns+-+Patterson+Ponds
http://example.com/cfp?waters=Surprise+-+Surprise+Lake
http://example.com/cfp?waters=Tempe+-+Evelyn+Hallman+Pond
http://example.com/cfp?waters=Tempe+-+Kiwanis+Lake
http://example.com/cfp?waters=Tempe+-+Tempe+Town+Lake
http://example.com/cfp?waters=Tucson+-+Kennedy+Lake
http://example.com/cfp?waters=Tucson+-+Lakeside+Lake
http://example.com/cfp?waters=Tucson+-+Silverbell+Lake
http://example.com/cfp?waters=Yuma+-+Fortuna+Lake
http://example.com/cfp?waters=Yuma+-+PAAC+Pond
http://example.com/cfp?waters=Yuma+-+West+Wetlands+Pond
http://example.com/winter
http://example.com/winter?waters=ASHURST+LAKE
http://example.com/winter?waters=BEAVER+CREEK+%28WET%29
http://example.com/winter?waters=BENDER%27S+POND
http://example.com/winter?waters=BUNCH+RESERVOIR
http://example.com/winter?waters=CANYON+CREEK
http://example.com/winter?waters=CATARACT+LAKE
http://example.com/winter?waters=CITY+RESERVOIR
http://example.com/winter?waters=CLUFF+POND
http://example.com/winter?waters=COUNCIL+AVENUE+POND
http://example.com/winter?waters=DANKWORTH+POND
http://example.com/winter?waters=DEADHORSE+LAKE+%28ST+PARK%29
http://example.com/winter?waters=EAST+VERDE+RIVER
http://example.com/winter?waters=FAIN+LAKE
http://example.com/winter?waters=FOOLS+HOLLOW+LAKE
http://example.com/winter?waters=FORTUNA+LAKE
http://example.com/winter?waters=Frances+Short
http://example.com/winter?waters=GOLDWATER+LAKE
http://example.com/winter?waters=GRAHAM+COUNTY+FAIRGROUNDS
http://example.com/winter?waters=HAIGLER+CREEK
http://example.com/winter?waters=KAIBAB+LAKE
http://example.com/winter?waters=LOWER+SALT+RIVER
http://example.com/winter?waters=LYNX+LAKE
http://example.com/winter?waters=MINGUS+LAKE
http://example.com/winter?waters=NELSON+RESERVOIR
http://example.com/winter?waters=OAK+CREEK
http://example.com/winter?waters=PAAC+POND
http://example.com/winter?waters=PARKER+%28LA+PAZ%29
http://example.com/winter?waters=PARKER+CANYON
http://example.com/winter?waters=PATAGONIA
http://example.com/winter?waters=PATAGONIA
http://example.com/winter?waters=PENA+BLANCA
http://example.com/winter?waters=RAINBOW+LAKE
http://example.com/winter?waters=REDONDO+LAKE
http://example.com/winter?waters=ROPER+LAKE
http://example.com/winter?waters=ROSE+CANYON+LAKE
http://example.com/winter?waters=SANTA+FE+LAKE
http://example.com/winter?waters=SCOTT+RESERVOIR
http://example.com/winter?waters=SHOW+LOW+LAKE
http://example.com/winter?waters=SILVER+CREEK
http://example.com/winter?waters=TONTO+CREEK
http://example.com/winter?waters=TUNNEL+RESERVOIR
http://example.com/winter?waters=VERDE+RIVER
http://example.com/winter?waters=WATSON+LAKE
http://example.com/winter?waters=WEST+CLEAR+CREEK
http://example.com/winter?waters=WEST+WETLANDS+POND
http://example.com/winter?waters=WOODLAND+RESERVOIR
http://example.com/winter?waters=YAVAPAI+LAKES
http://example.com/winter?waters=https%3A%2F%2Fwww.azgfd.com%2F
http://example.com/springsummer
`

	w := new(bytes.Buffer)
	server := &server{srv: srv, urlBase: "http://example.com"}
	server.writeSitemap(context.Background(), w)
	assert.Equal(t, expected, string(w.String()))
}
