package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/calvinmclean/stocker"
	"github.com/calvinmclean/stocker/internal/server"
	"github.com/calvinmclean/stocker/internal/transport"

	"github.com/urfave/cli/v2"
)

func main() {
	var debug, showNext, showLast, showAllStock, showAll bool
	var apiKey, programStr, addr, cacheDir, pushoverAppToken, pushoverRecipientToken string
	var cacheMaxAge time.Duration
	var waters []string
	app := &cli.App{
		Name: "stocker",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "debug", Usage: "enable debug logs", Destination: &debug},
			&cli.StringFlag{
				Name:        "api-key",
				Required:    true,
				Usage:       "Google API key to access Sheets",
				EnvVars:     []string{"API_KEY"},
				Destination: &apiKey,
			},
			&cli.DurationFlag{
				Name:        "cache-max-age",
				Usage:       "max time for cache expiration",
				Value:       24 * time.Hour,
				Destination: &cacheMaxAge,
			},
			&cli.StringFlag{
				Name:        "cache-dir",
				Usage:       "directory for disk cache. By default, in-memory cache is used",
				Destination: &cacheDir,
			},
		},
		DefaultCommand: "server",
		Commands: []*cli.Command{
			{
				Name:        "get",
				Description: "get info from the AZ GFD fish stocking schedule",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "next", Usage: "show next stocking time", Destination: &showNext},
					&cli.BoolFlag{Name: "last", Usage: "show recently-passed stocking time", Destination: &showLast},
					&cli.BoolFlag{Name: "all-stock", Usage: "show all stocking times in the schedule", Destination: &showAllStock},
					&cli.BoolFlag{Name: "all", Usage: "show full stocking schedule (include empty weeks)", Destination: &showAll},
					&cli.MultiStringFlag{
						Target: &cli.StringSliceFlag{
							Name:    "waters",
							Aliases: []string{"w"},
							Usage:   "",
						},
						Destination: &waters,
					},
					&cli.StringFlag{
						Name:        "program",
						Required:    true,
						Aliases:     []string{"p"},
						DefaultText: "CFP",
						Usage:       "AZ GFD Fishing program to search (CFP, Spring/Summer, or Winter)",
						Destination: &programStr,
					},
				},
				Action: func(c *cli.Context) error {
					program, err := stocker.ParseProgram(programStr)
					if err != nil {
						return err
					}

					rt := setupCacheControl(cacheMaxAge, cacheDir)
					if debug {
						rt = transport.Log(rt)
					}

					srv, err := stocker.NewService(apiKey, rt)
					if err != nil {
						return fmt.Errorf("error creating Sheets service: %w", err)
					}

					stockData, err := stocker.Get(srv, program, waters)
					if err != nil {
						return fmt.Errorf("error getting stocking data: %w", err)
					}

					for waterName, calendar := range stockData {
						fmt.Println(waterName)
						fmt.Println(calendar.DetailFormat(showAll, showAllStock, showNext, showLast))
					}
					return nil
				},
			},
			{
				Name: "server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "pushover-app-token",
						Usage:       "App token for Pushover notifications",
						Destination: &pushoverAppToken,
						EnvVars:     []string{"PUSHOVER_APP_TOKEN"},
					},
					&cli.StringFlag{
						Name:        "pushover-recipient-token",
						Usage:       "Recipient token for Pushover notifications",
						Destination: &pushoverRecipientToken,
						EnvVars:     []string{"PUSHOVER_RECIPIENT_TOKEN"},
					},
					&cli.StringFlag{
						Name:        "address",
						Usage:       "address to serve on",
						Destination: &addr,
						Value:       ":8080",
					},
				},
				Description: "run an HTTP server that responds with the AZ GFD fish stocking schedule",
				Action: func(ctx *cli.Context) error {
					rt := setupCacheControl(cacheMaxAge, cacheDir)
					if debug {
						rt = transport.Log(rt)
					}
					srv, err := stocker.NewService(apiKey, rt)
					if err != nil {
						return fmt.Errorf("error creating Sheets service: %w", err)
					}

					opts := []server.Option{}
					if pushoverAppToken != "" && pushoverRecipientToken != "" {
						opts = append(opts, server.WithPushoverClient(pushoverAppToken, pushoverRecipientToken))
					}

					return server.RunServer(addr, srv, opts...)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func setupCacheControl(maxAge time.Duration, dir string) http.RoundTripper {
	switch dir {
	case "":
		return transport.NewCacheControl(maxAge, nil)
	default:
		return transport.NewDiskCacheControl(dir, maxAge, nil)
	}
}
