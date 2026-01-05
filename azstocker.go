package azstocker

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	googleHTTP "google.golang.org/api/transport/http"
)

// override for setting time in tests
var getNow = time.Now

const (
	springSummerStockingSheetID   = "1S5wsDfGzEInV64UKjUPzexAe2KOO1KocfB4dJH7oVrs"
	springSummerStockingSheetName = "2025 Spring/Summer"

	winterStockingSheetID   = "1PZuTV-zi5vMdxaMSnGx6c-QxeQQm-6DRQJJPKAZDjZM"
	winterStockingSheetName = "2024-25 Winter"

	cfpStockingSheetID   = "1xJYPRrX2Gb7ACr6HxPB7mlsCw9K8NvClLfBIw7qjTcA"
	cfpStockingSheetName = "CFP Stocking Calendar Schedule"
)

const (
	CFProgram           Program = "cfp"
	WinterProgram       Program = "winter"
	SpringSummerProgram Program = "springsummer"
)

const (
	Catfish     Fish = "Catfish"
	Trout       Fish = "Trout"
	UnknownFish Fish = "Unknown"
	NoneFish    Fish = "None"
)

var azTime = time.FixedZone("AZ", -7*3600)

// Fish is the type of fish that is stocked
type Fish string

// ParseFish parses a string to a Fish type
func ParseFish(f string) Fish {
	switch strings.ToLower(f) {
	case "x", "t":
		return Trout
	case "c":
		return Catfish
	case "":
		return NoneFish
	default:
		return UnknownFish
	}
}

// Program is an enum type for AZ GFD stocking programs: cfp (community fishing program), winter,
// spring, and summer (spring/summer are the same)
type Program string

// ParseProgram parses a string to return a valid Program
func ParseProgram(p string) (Program, error) {
	switch strings.ToLower(p) {
	case string(CFProgram):
		return CFProgram, nil
	case string(WinterProgram):
		return WinterProgram, nil
	case string(SpringSummerProgram), "spring", "summer":
		return SpringSummerProgram, nil
	default:
		return "", errors.New("unknown program")
	}
}

// Week represents a date on the calendar and shows stocking data for that week range
type Week struct {
	Month time.Month
	Day   int
	Year  int
	Stock Fish
}

// Time creates a time.Time from the Year, Month, and Date of stocking
func (s Week) Time() time.Time {
	return time.Date(s.Year, s.Month, s.Day, 0, 0, 0, 0, azTime)
}

func (s Week) HumanTime() string {
	return humanize.RelTime(s.Time(), getNow(), "ago", "from now")
}

// String formats the Week to show the date and stocking data
func (s Week) String() string {
	if s.Year == 0 && s.Day == 0 {
		return "No Data"
	}
	return fmt.Sprintf("%d %s %d: %q", s.Year, s.Month.String(), s.Day, s.Stock)
}

// Calendar is and ordered list of Weeks and shows all available stocking data for a specific water
type Calendar struct {
	WaterName string
	Data      []Week
}

// String formats the Calendar and excludes non-stocked dates
func (s Calendar) String() string {
	return s.Format(false)
}

// Format all dates in the Calendar. If hideEmpty is set, it will exclude non-stocking days
func (s Calendar) Format(hideEmpty bool) string {
	var sb strings.Builder
	for _, data := range s.Data {
		if hideEmpty && data.Stock == NoneFish {
			continue
		}
		sb.WriteString(data.String())
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// DetailFormat creates string with detailed explanation of the Calendar and accepts a few boolean controls
func (s Calendar) DetailFormat(showAll, showAllStock, next, last bool) string {
	var sb strings.Builder

	// if all are unset, default to just printing scheduled times
	if !showAll && !showAllStock && !next && !last {
		sb.WriteString(s.Format(false))
		return sb.String()
	}

	if showAll {
		sb.WriteString(s.Format(false))
		sb.WriteString("\n")
	} else if showAllStock {
		sb.WriteString(s.Format(true))
		sb.WriteString("\n")
	}

	if last {
		sb.WriteString("Last: ")
		sb.WriteString(s.Last().String())
		sb.WriteString("\n")
	}
	if next {
		sb.WriteString("Next:")
		sb.WriteString(s.Next().String())
	}

	return sb.String()
}

// Next returns the closest upcoming StockingData based on the current time
func (s Calendar) Next() Week {
	now := getNow().In(azTime)

	for _, data := range s.Data {
		if data.Stock == NoneFish || data.Stock == UnknownFish {
			continue
		}
		if data.Time().After(now) {
			return data
		}
	}

	return Week{}
}

// Last returns the most recent StockingData based on the current time
func (s Calendar) Last() Week {
	now := getNow().In(azTime)

	for _, data := range slices.Backward(s.Data) {
		if data.Stock == NoneFish {
			continue
		}
		if data.Time().Before(now) {
			return data
		}
	}

	return Week{}
}

// StockingData is a slice of stocking Calendars for different waters
type StockingData []Calendar

// Sort allows sorting the data by a compare function and will sort alphabetically if compare is equal
func (s StockingData) Sort(compare func(Calendar, Calendar) int) {
	slices.SortFunc(s, func(a, b Calendar) int {
		comp := compare(a, b)
		if comp == 0 {
			comp = strings.Compare(a.WaterName, b.WaterName)
		}
		return comp
	})
}

// SortNext sorts by closest upcoming stocking dates
func (s StockingData) SortNext() {
	s.Sort(func(c1, c2 Calendar) int {
		c1Next := c1.Next()
		c2Next := c2.Next()
		if c1Next.Year == 0 {
			return 1
		}
		if c2Next.Year == 0 {
			return -1
		}
		return c1Next.Time().Compare(c2Next.Time())
	})
}

// SortLast sorts by most-recently stocked waters
func (s StockingData) SortLast() {
	// sort reverse since we are looking for largest time first
	s.Sort(func(c1, c2 Calendar) int {
		c1Next := c1.Last()
		c2Next := c2.Last()
		if c1Next.Year == 0 {
			return 1
		}
		if c2Next.Year == 0 {
			return -1
		}
		return c2Next.Time().Compare(c1Next.Time())
	})
}

type sheet struct {
	srv           *sheets.Service
	spreadsheetID string
	sheetName     string

	// A1 notation range to get water name and schedule
	scheduleRange string
	// A1 notation range to get dates
	dateRange string

	// winter schedule has a column deleted from the sheet, but it shows up as empty in the raw data
	skipDataCol int
}

// create a new Sheet depending on the required program
func newSheet(srv *sheets.Service, program Program) *sheet {
	switch program {
	case CFProgram:
		return &sheet{
			srv:           srv,
			spreadsheetID: cfpStockingSheetID,
			sheetName:     cfpStockingSheetName,
			scheduleRange: "A11:Z",
			dateRange:     "B8:9",
			skipDataCol:   -1,
		}
	case WinterProgram:
		return &sheet{
			srv:           srv,
			spreadsheetID: winterStockingSheetID,
			sheetName:     winterStockingSheetName,
			scheduleRange: "A9:AD",
			dateRange:     "B4:5",
			skipDataCol:   5,
		}
	case SpringSummerProgram:
		return &sheet{
			srv:           srv,
			spreadsheetID: springSummerStockingSheetID,
			sheetName:     springSummerStockingSheetName,
			scheduleRange: "A9:AD",
			dateRange:     "B4:5",
			skipDataCol:   5,
		}
	default:
		return nil
	}
}

func (s *sheet) getDataForWaters(waterNames []string) (StockingData, error) {
	lowerCaseWaterNames := []string{}
	for _, w := range waterNames {
		lowerCaseWaterNames = append(lowerCaseWaterNames, strings.ToLower(w))
	}

	stockingCalendar, err := s.initializeCalendar()
	if err != nil {
		return nil, fmt.Errorf("error initializing calendar: %w", err)
	}

	data, err := s.getStockingData(stockingCalendar, lowerCaseWaterNames)
	if err != nil {
		return nil, fmt.Errorf("error finding water rows: %w", err)
	}
	return data, nil
}

// getStockingData parses a sheet to populate the provided Calendar dates with stocking data for specified waters.
func (s *sheet) getStockingData(stockingCalendar Calendar, waterNames []string) (StockingData, error) {
	readRange := fmt.Sprintf("%s!%s", s.sheetName, s.scheduleRange)
	resp, err := s.srv.Spreadsheets.Values.Get(s.spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting data from sheet: %w", err)
	}

	result := []Calendar{}
	for _, row := range resp.Values {
		if len(row) < 2 {
			continue
		}

		waterName := cellAsString(row[0])
		if waterName == "" {
			continue
		}
		if len(waterNames) > 0 && !slices.Contains(waterNames, strings.ToLower(waterName)) {
			continue
		}

		data, err := s.getDataFromRow(row[1:], stockingCalendar)
		if err != nil {
			// TODO: This is not best practice...
			log.Printf("error getting data for row %q: %v", waterName, err)
			continue
		}
		data.WaterName = waterName
		result = append(result, data)
	}

	return result, nil
}

// getDataFromRow parses a row and adds stocking data to the provided Calendar dates
func (s *sheet) getDataFromRow(row []any, stockingCalendar Calendar) (Calendar, error) {
	// if s.skipDataCol is set, then we will need to skip a col eventually and need to account for this
	// when appending empty data
	skippedRows := 0
	if s.skipDataCol >= 0 {
		skippedRows = 1
	}
	// empty trailing cols are trimmed, so we append until we have the correct number of cols
	for len(row)-skippedRows < len(stockingCalendar.Data) {
		row = append(row, "")
	}
	if len(stockingCalendar.Data) != len(row)-skippedRows {
		return Calendar{}, fmt.Errorf("dates and stock rows don't match: %d != %d\n", len(stockingCalendar.Data), len(row))
	}

	result := Calendar{}
	skippedRows = 0
	for i, stock := range row {
		if i == s.skipDataCol {
			skippedRows = 1
			continue
		}

		dateItem := stockingCalendar.Data[i-skippedRows]
		dateItem.Stock = ParseFish(cellAsString(stock))

		result.Data = append(result.Data, dateItem)
	}
	return result, nil
}

// initializeCalendar parses the date rows of the Sheet to initialize the Calendar dates
func (s *sheet) initializeCalendar() (Calendar, error) {
	readRange := fmt.Sprintf("%s!%s", s.sheetName, s.dateRange)
	resp, err := s.srv.Spreadsheets.Values.Get(s.spreadsheetID, readRange).Do()
	if err != nil {
		return Calendar{}, fmt.Errorf("error getting data from sheet: %w", err)
	}

	if len(resp.Values) != 2 {
		return Calendar{}, fmt.Errorf("expected 2 rows but got %d", len(resp.Values))
	}

	monthCells := resp.Values[0]
	dayCells := resp.Values[1]

	months := []time.Time{}
	for _, month := range nonEmptyCells(monthCells) {
		m := parseMonth(month)
		if m != nil {
			months = append(months, *m)
		}
	}

	result := Calendar{}
	year := chooseCurrentYear(months)
	monthIndex := 0
	prevDay := -1
	for _, date := range nonEmptyCells(dayCells) {
		// split cell for CFP schedule which is formatted like 7-11
		day, err := strconv.Atoi(strings.Split(date, "-")[0])
		if err != nil {
			continue
		}

		// When the current day is less than the previous, we are in a new month
		if day < prevDay {
			monthIndex++

			// check for year rollover if it's January
			if isNewYear(months, monthIndex) {
				year++
			}
		}
		prevDay = day
		if monthIndex >= len(months) {
			return Calendar{}, fmt.Errorf("index out of range: %d", monthIndex)
		}

		useYear := months[monthIndex].Year()
		if useYear == 0 {
			useYear = year
		}
		result.Data = append(result.Data, Week{
			Year:  useYear,
			Month: months[monthIndex].Month(),
			Day:   day,
		})
	}

	return result, nil
}

// chooseCurrentYear helps decide what year it is. If a parsed year exists in a month use that. Othwise, determine
// best guess based on current month range
func chooseCurrentYear(months []time.Time) int {
	if len(months) > 0 && months[0].Year() != 0 {
		return months[0].Year()
	}

	if len(months) < 2 {
		return getNow().Year()
	}

	month1, month2 := months[0].Month(), months[len(months)-1].Month()

	// if first month is > last, then it might be starting with previous year
	if month1 > month2 {
		return getNow().Year() - 1
	}

	return getNow().Year()
}

// NewService is a shortcut for creating a sheets.Service using an API key and a custom HTTP RoundTripper.
// If RoundTripper is not provided, http.DefaultTransport will be used
func NewService(apiKey string, rt http.RoundTripper) (*sheets.Service, error) {
	transport, err := googleHTTP.NewTransport(context.Background(), rt, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error creating transport: %w", err)
	}
	client := &http.Client{Transport: transport}

	googleClient, _, err := googleHTTP.NewClient(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	srv, err := sheets.NewService(context.Background(), option.WithHTTPClient(googleClient))
	if err != nil {
		return nil, fmt.Errorf("error creating service: %w", err)
	}

	return srv, nil
}

// Get will parse the Google Sheet for the specified Program. If waters are provided, it will only return data
// for these waters. Otherwise, it provides for all
func Get(srv *sheets.Service, program Program, waters []string) (StockingData, error) {
	sheet := newSheet(srv, program)
	if sheet == nil {
		return nil, fmt.Errorf("unable to initialize sheet for program %q", program)
	}

	stockData, err := sheet.getDataForWaters(waters)
	if err != nil {
		return nil, err
	}
	return stockData, nil
}

func isNewYear(months []time.Time, i int) bool {
	if i >= len(months) {
		return false
	}
	return months[i].Month() == time.January && i > 0 && months[i-1].Month() == time.December
}

func nonEmptyCells(cells []any) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, cell := range cells {
			cellStr := cellAsString(cell)
			if cellStr == "" {
				continue
			}
			shouldContinue := yield(i, cellStr)
			if !shouldContinue {
				return
			}
		}
	}
}

func cellAsString(cell any) string {
	cellStr, ok := cell.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(cellStr)
}

var monthMap = map[string]time.Month{
	"january":  time.January,
	"february": time.February,
	// Typo in CFP sheet
	"feburary":  time.February,
	"march":     time.March,
	"april":     time.April,
	"may":       time.May,
	"june":      time.June,
	"july":      time.July,
	"august":    time.August,
	"september": time.September,
	"october":   time.October,
	"november":  time.November,
	"december":  time.December,
}

// parseMonth parses a string like "OCTOBER 2025" or "OCTOBER" and returns a Time holding the month/year
func parseMonth(in string) *time.Time {
	parts := strings.Split(in, " ")
	if len(parts) == 0 {
		return nil
	}

	month, ok := monthMap[strings.ToLower(parts[0])]
	if !ok {
		return nil
	}

	// Set day to 15 to avoid rolling over to previous month somehow
	date := time.Date(0, month, 15, 0, 0, 0, 0, time.UTC)
	if len(parts) == 2 {
		year, err := strconv.Atoi(parts[1])
		if err == nil {
			date = date.AddDate(year, 0, 0)
		}
	}

	return &date
}
