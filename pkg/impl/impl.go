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

type PatternRule struct {
	Name     string
	Format   func(parts []string) string
	Score    float64
	Reason   string
	MinParts int
}

func getMiddleInitials(parts []string) string {
	var m strings.Builder
	for i := 1; i < len(parts)-1; i++ {
		if len(parts[i]) > 0 {
			m.WriteString(string(parts[i][0]))
		}
	}
	return m.String()
}

var patternRules = []PatternRule{
	{Name: "first.last", MinParts: 2, Score: 0.90, Reason: "Enterprise standard I", Format: func(p []string) string { return p[0] + "." + p[len(p)-1] }},
	{Name: "flast", MinParts: 2, Score: 0.75, Reason: "Legacy corporate format", Format: func(p []string) string { return string(p[0][0]) + p[len(p)-1] }},
	{Name: "firstlast", MinParts: 2, Score: 0.60, Reason: "Common for startups", Format: func(p []string) string { return p[0] + p[len(p)-1] }},
	{Name: "f.last", MinParts: 2, Score: 0.50, Reason: "Finance/Academia variant", Format: func(p []string) string { return string(p[0][0]) + "." + p[len(p)-1] }},
	{Name: "first", MinParts: 1, Score: 0.45, Reason: "Small startups (<20 employees)", Format: func(p []string) string { return p[0] }},
	{Name: "firstl", MinParts: 2, Score: 0.40, Reason: "First name, last initial (Pre-collision resolution)", Format: func(p []string) string { return p[0] + string(p[len(p)-1][0]) }},
	{Name: "first_last", MinParts: 2, Score: 0.35, Reason: "Occasional alternative", Format: func(p []string) string { return p[0] + "_" + p[len(p)-1] }},

	{Name: "fmlast", MinParts: 3, Score: 0.65, Reason: "Common corporate (with middle initial)", Format: func(p []string) string { return string(p[0][0]) + getMiddleInitials(p) + p[len(p)-1] }},
	{Name: "f.m.last", MinParts: 3, Score: 0.55, Reason: "Segmented middle initials", Format: func(p []string) string {
		var inits []string
		for i := 1; i < len(p)-1; i++ {
			inits = append(inits, string(p[i][0]))
		}
		return string(p[0][0]) + "." + strings.Join(inits, ".") + "." + p[len(p)-1]
	}},
	{Name: "first.m.last", MinParts: 3, Score: 0.45, Reason: "Full first, initial middle", Format: func(p []string) string { return p[0] + "." + getMiddleInitials(p) + "." + p[len(p)-1] }},
	{Name: "first.middle.last", MinParts: 3, Score: 0.40, Reason: "Highly specific, rarely used", Format: func(p []string) string { return strings.Join(p, ".") }},

	{Name: "first.last1", MinParts: 2, Score: 0.15, Reason: "Enterprise collision (1st duplicate)", Format: func(p []string) string { return p[0] + "." + p[len(p)-1] + "1" }},
	{Name: "flast1", MinParts: 2, Score: 0.12, Reason: "Legacy corporate collision", Format: func(p []string) string { return string(p[0][0]) + p[len(p)-1] + "1" }},
	{Name: "first1", MinParts: 1, Score: 0.08, Reason: "Startup first name collision", Format: func(p []string) string { return p[0] + "1" }},
	{Name: "first.last2", MinParts: 2, Score: 0.05, Reason: "Enterprise collision (2nd duplicate)", Format: func(p []string) string { return p[0] + "." + p[len(p)-1] + "2" }},
	{Name: "flast2", MinParts: 2, Score: 0.04, Reason: "Legacy corporate collision (2nd)", Format: func(p []string) string { return string(p[0][0]) + p[len(p)-1] + "2" }},

	{Name: "last.first", MinParts: 2, Score: 0.20, Reason: "Govt domains", Format: func(p []string) string { return p[len(p)-1] + "." + p[0] }},
	{Name: "last", MinParts: 2, Score: 0.10, Reason: "Generic fallback", Format: func(p []string) string { return p[len(p)-1] }},
}

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

func GeneratePermutations(fullName, domain string) []Permutation {
	dom := strings.ToLower(strings.TrimSpace(domain))

	// Tokenize the name into parts, dropping extra spaces
	rawParts := strings.Fields(strings.ToLower(strings.TrimSpace(fullName)))
	var parts []string
	for _, rp := range rawParts {
		// Clean out non-alpha characters from names if needed
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

	// Track generated emails to prevent duplicates (e.g. if name is just "kuria")
	seen := make(map[string]bool)

	for _, rule := range patternRules {
		if len(parts) < rule.MinParts {
			continue
		}

		user := rule.Format(parts)
		email := fmt.Sprintf("%s@%s", user, dom)

		if seen[email] {
			continue
		}
		seen[email] = true

		results = append(results, Permutation{
			Email:        email,
			Pattern:      rule.Name,
			Score:        rule.Score,
			Plausibility: getPlausibility(rule.Score),
			Reason:       rule.Reason,
			SMTPStatus:   "Unverified",
		})
	}

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

	sort.Slice(mxs, func(i, j int) bool { return mxs[i].Pref < mxs[j].Pref })
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
