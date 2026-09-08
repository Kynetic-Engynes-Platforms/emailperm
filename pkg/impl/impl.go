package impl

import (
	"fmt"
	"net"
	"net/smtp"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
)

type Permutation struct {
	Email        string  `json:"email"`
	Pattern      string  `json:"pattern"`
	Score        float64 `json:"score"`
	Plausibility string  `json:"plausibility"`
	Reason       string  `json:"reason"`
	SMTPStatus   string  `json:"smtp_status,omitempty"`
}

type NameState int

const (
	StateFull NameState = iota
	StateInitial
	StateOmit
)

var separators = []string{"", ".", "-", "_"}
var suffixes = []string{"", "1", "2"}

func getPlausibility(score float64) string {
	switch {
	case score >= 0.85:
		return "Very High"
	case score >= 0.70:
		return "High"
	case score >= 0.45:
		return "Medium"
	default:
		return "Low"
	}
}

func ColorizeScore(score float64) string {
	s := fmt.Sprintf("%.2f", score)
	if score >= 0.70 {
		return text.FgGreen.Sprint(s)
	}
	if score >= 0.45 {
		return text.FgYellow.Sprint(s)
	}
	return text.FgRed.Sprint(s)
}

func ColorizeSMTP(status string) string {
	switch {
	case strings.Contains(status, "250 OK"):
		return text.FgGreen.Sprint(status)
	case strings.Contains(status, "Catch-All"):
		return text.FgYellow.Sprint(status)
	case strings.Contains(status, "Invalid"):
		return text.FgRed.Sprint(status)
	default:
		return text.FgHiBlack.Sprint(status)
	}
}

func generateEmailsFromStates(parts []string, states []NameState, results *[]Permutation, domain string, seen map[string]bool) {
	for _, sep := range separators {
		for _, suffix := range suffixes {
			var emailParts []string
			var patternDesc []string
			score := 0.40 // Base score threshold

			for i, state := range states {
				switch state {
				case StateFull:
					emailParts = append(emailParts, parts[i])
					patternDesc = append(patternDesc, "Full")
					score += 0.20 // Reward full name utilization
				case StateInitial:
					emailParts = append(emailParts, string(parts[i][0]))
					patternDesc = append(patternDesc, "Initial")
					score += 0.10 // Minor reward for initials
				case StateOmit:
					patternDesc = append(patternDesc, "Omitted")
				}
			}

			// Adjust scores based on IT policy likelihoods
			if sep == "." {
				score += 0.15 // Enterprise standard
			} else if sep == "" {
				score += 0.05 // Common startup standard
			} else {
				score -= 0.10 // Hyphens/underscores are less common
			}

			if suffix != "" {
				score -= 0.30 // Penalize collisions as they are edge cases
			}

			// Construct email
			userPrefix := strings.Join(emailParts, sep) + suffix
			email := fmt.Sprintf("%s@%s", userPrefix, domain)
			reason := strings.Join(patternDesc, "-") + " w/ separator '" + sep + "'"

			// Cap score at 0.99 for realism
			if score > 0.99 {
				score = 0.99
			}
			if score < 0.01 {
				score = 0.01
			}

			if !seen[email] {
				seen[email] = true
				*results = append(*results, Permutation{
					Email:        email,
					Pattern:      "Dynamic",
					Score:        score,
					Plausibility: getPlausibility(score),
					Reason:       reason,
					SMTPStatus:   "Unverified",
				})
			}
		}
	}
}

func buildCombinations(parts []string, index int, current []NameState, results *[]Permutation, domain string, seen map[string]bool) {
	if index == len(parts) {
		generateEmailsFromStates(parts, current, results, domain, seen)
		return
	}

	buildCombinations(parts, index+1, append(current, StateFull), results, domain, seen)
	buildCombinations(parts, index+1, append(current, StateInitial), results, domain, seen)

	// Only allow omitting middle names
	isMiddleName := index > 0 && index < len(parts)-1
	if isMiddleName {
		buildCombinations(parts, index+1, append(current, StateOmit), results, domain, seen)
	}
}

// --- Main Exported Functions ---

func GeneratePermutations(fullName, domain string) []Permutation {
	dom := strings.ToLower(strings.TrimSpace(domain))

	// Pre-process: Replace hyphens with spaces to split compound names (e.g., Mary-Jane -> Mary Jane)
	spacedName := strings.ReplaceAll(fullName, "-", " ")
	rawParts := strings.Fields(strings.ToLower(strings.TrimSpace(spacedName)))

	var parts []string
	for _, rp := range rawParts {
		// Clean out non-alpha characters (O'Connor -> oconnor)
		clean := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return r
			}
			return -1
		}, rp)
		if len(clean) > 0 {
			parts = append(parts, clean)
		}
	}

	var results []Permutation
	if len(parts) == 0 {
		return results
	}

	seen := make(map[string]bool)
	buildCombinations(parts, 0, []NameState{}, &results, dom, seen)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

func CheckSMTP(domain string, emails []string, tracker *progress.Tracker) (map[string]string, bool, error) {
	mxs, err := net.LookupMX(domain)
	if err != nil || len(mxs) == 0 {
		return nil, false, fmt.Errorf("no MX records found for %s", domain)
	}

	sort.Slice(mxs, func(i, j int) bool {
		return mxs[i].Pref < mxs[j].Pref
	})

	mxHost := strings.TrimSuffix(mxs[0].Host, ".")
	conn, err := net.DialTimeout("tcp", mxHost+":25", 5*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("MX connection failed (ISP port 25 block?): %v", err)
	}

	client, err := smtp.NewClient(conn, mxHost)
	if err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	defer client.Close()

	if err := client.Hello("validator.local"); err != nil {
		return nil, false, fmt.Errorf("HELO failed: %v", err)
	}
	if err := client.Mail("verify@validator.local"); err != nil {
		return nil, false, fmt.Errorf("MAIL FROM failed: %v", err)
	}

	catchAll := false
	randomEmail := fmt.Sprintf("invalid-test-%d@%s", time.Now().UnixNano(), domain)
	if err := client.Rcpt(randomEmail); err == nil {
		catchAll = true
	}

	results := make(map[string]string)
	for _, email := range emails {
		if catchAll {
			results[email] = "Valid (Catch-All Domain)"
		} else {
			if err := client.Rcpt(email); err == nil {
				results[email] = "Valid (250 OK)"
			} else {
				results[email] = "Invalid / Rejected"
			}
		}
		if tracker != nil {
			tracker.Increment(1)
			time.Sleep(time.Millisecond * 50)
		}
	}

	_ = client.Quit()
	return results, catchAll, nil
}
