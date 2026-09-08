package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Kynetic-Engynes-Platforms/emailperm/pkg/impl"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "emailperm",
		Usage: "Generate and verify plausible email permutations (supports multi-part names)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Full name (e.g. 'Kuria Mwangi Weru')", Required: true},
			&cli.StringFlag{Name: "domain", Aliases: []string{"d"}, Usage: "Target domain (e.g. kyneticengynes.com)", Required: true},
			&cli.BoolFlag{Name: "verify", Aliases: []string{"v"}, Usage: "Perform active SMTP MX verification"},
			&cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Usage: "Output as JSON"},
		},

		Action: func(ctx context.Context, c *cli.Command) error {
			fullName := c.String("name")
			domain := c.String("domain")
			verify := c.Bool("verify")
			asJSON := c.Bool("json")

			permutations := impl.GeneratePermutations(fullName, domain)

			if len(permutations) == 0 {
				fmt.Fprintln(os.Stderr, text.FgRed.Sprint("[!] Error: Name provided did not contain valid alphabetical characters."))
				return nil
			}

			if verify {
				var emails []string
				for _, p := range permutations {
					emails = append(emails, p.Email)
				}

				var tracker *progress.Tracker
				var pw progress.Writer

				if !asJSON {
					pw = progress.NewWriter()
					pw.SetAutoStop(true)
					pw.SetTrackerLength(25)
					pw.SetMessageWidth(30)
					pw.SetStyle(progress.StyleDefault)
					pw.Style().Colors = progress.StyleColorsExample
					pw.Style().Options.TimeInProgressPrecision = time.Millisecond

					go pw.Render()

					tracker = &progress.Tracker{
						Message: fmt.Sprintf("Verifying %s...", domain),
						Total:   int64(len(emails)),
						Units:   progress.UnitsDefault,
					}
					pw.AppendTracker(tracker)
				}

				results, catchAll, err := impl.CheckSMTP(domain, emails, tracker)

				if tracker != nil {
					if err != nil {
						tracker.MarkAsErrored()
					} else {
						tracker.MarkAsDone()
					}
					time.Sleep(time.Millisecond * 100)
				}

				if err != nil && !asJSON {
					fmt.Fprintf(os.Stderr, "\n%s %v\n", text.FgRed.Sprint("[!] SMTP Check Error:"), err)
				} else {
					if catchAll && !asJSON {
						fmt.Fprintf(os.Stderr, "\n%s Catch-all domain detected. Server accepts all prefixes.\n", text.FgYellow.Sprint("[!]"))
					}
					for i, p := range permutations {
						permutations[i].SMTPStatus = results[p.Email]
					}
				}
			}

			if asJSON {
				output, _ := json.MarshalIndent(permutations, "", "  ")
				fmt.Println(string(output))
				return nil
			}

			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row{"Email", "Pattern", "Score", "SMTP Status", "Reason"})
			t.Style().Format.Header = text.FormatUpper
			t.Style().Color.Header = text.Colors{text.Bold}

			for _, p := range permutations {
				t.AppendRow(table.Row{
					text.FgHiCyan.Sprint(p.Email),
					p.Pattern,
					impl.ColorizeScore(p.Score),
					impl.ColorizeSMTP(p.SMTPStatus),
					p.Reason,
				})
			}

			t.SetStyle(table.StyleRounded)
			fmt.Println()
			t.Render()

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
