package stocker

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

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	googleHTTP "google.golang.org/api/transport/http"
)

const (
	springSummerStockingSheetID   = "1S5wsDfGzEInV64UKjUPzexAe2KOO1KocfB4dJH7oVrs"
	springSummerStockingSheetName = "2024 Spring/Summer"

	winterStockingSheetID   = "1PZuTV-zi5vMdxaMSnGx6c-QxeQQm-6DRQJJPKAZDjZM"
	winterStockingSheetName = "2024-25 Winter"

	cfpStockingSheetID   = "1xJYPRrX2Gb7ACr6HxPB7mlsCw9K8NvClLfBIw7qjTcA"
	cfpStockingSheetName = "CFP Stocking Calendar Schedule"
)

const (
	CFProgram           Program = "cfp"
	WinterProgram       Program = "winter"
	SpringSummerProgram Program = "spring/summer"
)

var azTime = time.FixedZone("AZ", -7)

// Program is an enum type for AZ GFD stocking programs: cfp (community fishing program), winter,
// spring, and summer (spring/summer are the same)
type Program string

// ParseProgram parses a string to return a valid Program
func ParseProgram(p string) (Program, error) {
	switch strings.ToLower(p) {
	case string(CFProgram):
		return CFProgram, nil
	case string(WinterProgram):
		return CFProgram, nil
	case "spring", "summer":
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
	Stock string
}

// Time creates a time.Time from the Year, Month, and Date of stocking
func (s Week) Time() time.Time {
	return time.Date(s.Year, s.Month, s.Day, 0, 0, 0, 0, azTime)
}

// String formats the Week to show the date and stocking data
func (s Week) String() string {
	if s.Year == 0 && s.Day == 0 {
		return "No Data"
	}
	return fmt.Sprintf("%d %s %d: %q", s.Year, s.Month.String(), s.Day, s.Stock)
}

// Calendar is and ordered list of Weeks and shows all available stocking data for a specific water
type Calendar []Week

// String formats the Calendar and excludes non-stocked dates
func (s Calendar) String() string {
	return s.Format(false)
}

// Format all dates in the Calendar. If hideEmpty is set, it will exclude non-stocking days
func (s Calendar) Format(hideEmpty bool) string {
	var sb strings.Builder
	for _, data := range s {
		if hideEmpty && data.Stock == "" {
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
	now := time.Now().In(azTime)

	for _, data := range s {
		if data.Stock == "" {
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
	now := time.Now().In(azTime)

	for _, data := range slices.Backward(s) {
		if data.Stock == "" {
			continue
		}
		if data.Time().Before(now) {
			return data
		}
	}

	return Week{}
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

func (s *sheet) getDataForWaters(waterNames []string) (map[string]Calendar, []string, error) {
	lowerCaseWaterNames := []string{}
	for _, w := range waterNames {
		lowerCaseWaterNames = append(lowerCaseWaterNames, strings.ToLower(w))
	}

	stockingCalendar, err := s.initializeCalendar()
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing calendar: %w", err)
	}

	data, allWaterNames, err := s.getStockingData(stockingCalendar, lowerCaseWaterNames)
	if err != nil {
		return nil, nil, fmt.Errorf("error finding water rows: %w", err)
	}
	return data, allWaterNames, nil
}

// getStockingData parses a sheet to populate the provided Calendar dates with stocking data for specified waters.
// Also returns a list of all water names in the sheet
func (s *sheet) getStockingData(stockingCalendar Calendar, waterNames []string) (map[string]Calendar, []string, error) {
	readRange := fmt.Sprintf("%s!%s", s.sheetName, s.scheduleRange)
	resp, err := s.srv.Spreadsheets.Values.Get(s.spreadsheetID, readRange).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting data from sheet: %w", err)
	}

	allWaterNames := []string{}
	result := map[string]Calendar{}
	for _, row := range resp.Values {
		if len(row) < 2 {
			continue
		}

		waterName := cellAsString(row[0])
		if waterName == "" {
			continue
		}
		allWaterNames = append(allWaterNames, waterName)
		if len(waterNames) > 0 && !slices.Contains(waterNames, strings.ToLower(waterName)) {
			continue
		}

		result[waterName], err = s.getDataFromRow(row[1:], stockingCalendar)
		if err != nil {
			// TODO: This is not best practice...
			log.Printf("error getting data for row %q: %v", waterName, err)
			continue
		}
	}

	return result, allWaterNames, nil
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
	for len(row)-skippedRows < len(stockingCalendar) {
		row = append(row, "")
	}
	if len(stockingCalendar) != len(row)-skippedRows {
		return nil, fmt.Errorf("dates and stock rows don't match: %d != %d\n", len(stockingCalendar), len(row))
	}

	result := Calendar{}
	skippedRows = 0
	for i, stock := range row {
		if i == s.skipDataCol {
			skippedRows = 1
			continue
		}

		dateItem := stockingCalendar[i-skippedRows]
		dateItem.Stock = cellAsString(stock)

		result = append(result, dateItem)
	}
	return result, nil
}

// initializeCalendar parses the date rows of the Sheet to initialize the Calendar dates
func (s *sheet) initializeCalendar() (Calendar, error) {
	readRange := fmt.Sprintf("%s!%s", s.sheetName, s.dateRange)
	resp, err := s.srv.Spreadsheets.Values.Get(s.spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting data from sheet: %w", err)
	}

	if len(resp.Values) != 2 {
		return nil, fmt.Errorf("expected 2 rows but got %d", len(resp.Values))
	}

	monthCells := resp.Values[0]
	dayCells := resp.Values[1]

	months := []time.Month{}
	for _, month := range nonEmptyCells(monthCells) {
		m := parseMonth(month)
		if m != nil {
			months = append(months, *m)
		}
	}

	result := Calendar{}
	year := time.Now().Year()
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

		result = append(result, Week{
			Year:  year,
			Month: months[monthIndex],
			Day:   day,
		})
	}

	return result, nil
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
// for these waters. Otherwise, it provides for all. This returns a map of water name to the Calendar data. It
// also returns a list of all waters in the Sheet
func Get(srv *sheets.Service, program Program, waters []string) (map[string]Calendar, []string, error) {
	sheet := newSheet(srv, program)
	if sheet == nil {
		return nil, nil, fmt.Errorf("unable to initialize sheet for program %q", program)
	}

	stockData, allWaterNames, err := sheet.getDataForWaters(waters)
	if err != nil {
		return nil, nil, err
	}
	return stockData, allWaterNames, nil
}

func isNewYear(months []time.Month, i int) bool {
	return months[i] == time.January && i > 0 && months[i-1] == time.December
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
	"january":   time.January,
	"february":  time.February,
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

func parseMonth(in string) *time.Month {
	in = strings.TrimSuffix(in, " 2024")
	result, ok := monthMap[strings.ToLower(in)]
	if !ok {
		return nil
	}
	return &result
}
