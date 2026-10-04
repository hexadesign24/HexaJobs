// Package security provides local text heuristics, not identity verification.
package security

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"hexajobs.dev/hexajobs-cli/internal/models"
)

type rule struct {
	pattern        *regexp.Regexp
	flag, severity string
}

var rules = []rule{
	{regexp.MustCompile(`\btelegram[\s_-]+only\b`), "Telegram-only recruitment", models.RiskRed},
	{regexp.MustCompile(`\bupfront[\s_-]+(?:registration[\s_-]+)?fee\b`), "Upfront fee requested", models.RiskRed},
	{regexp.MustCompile(`\b(?:bank[\s_-]+account[\s_-]+rental|rent[\s_-]+(?:your[\s_-]+)?bank[\s_-]+account)\b`), "Bank account rental", models.RiskRed},
	{regexp.MustCompile(`\bwhats\s*app[\s_-]+interview\b`), "WhatsApp-only interview channel needs review", models.RiskYellow},
	{regexp.MustCompile(`\b(?:seed[\s_-]+phrase|private[\s_-]+key)\b`), "Private wallet credentials mentioned", models.RiskYellow},
	{regexp.MustCompile(`\b(?:pay[\s_-]+to[\s_-]+apply|deposit[\s_-]+required|biaya[\s_-]+pendaftaran|sewa[\s_-]+rekening)\b`), "Payment or account access requested", models.RiskRed},
}

// ValidateJobListing preserves a stronger engine risk assessment. GREEN means
// no rule matched, not that a company or smart contract was independently audited.
func ValidateJobListing(job models.JobListing) (status string, flags []string) {
	status = models.RiskGreen
	flags = []string{}
	if job.RiskStatus == models.RiskRed {
		status = models.RiskRed
		flags = append(flags, "Engine flagged high risk")
	} else if job.RiskStatus == models.RiskYellow {
		status = models.RiskYellow
		flags = append(flags, "Source verification incomplete")
	}
	text := strings.ToLower(ansi.Strip(job.Title + " " + job.Company + " " + job.Description + " " + job.CompensationRaw))
	text = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	for _, r := range rules {
		if r.pattern.MatchString(text) {
			flags = append(flags, r.flag)
			if r.severity == models.RiskRed {
				status = models.RiskRed
			} else if status != models.RiskRed {
				status = models.RiskYellow
			}
		}
	}
	// Use the engine's normalized amount only for an explicitly hourly listing.
	if regexpDataEntry.MatchString(text) && job.CompensationPeriod == "hour" && job.CompensationUSD > 250 {
		status = models.RiskRed
		flags = append(flags, "Basic data entry advertised above USD 250/hour")
	}
	if job.IsMock {
		flags = append(flags, "Demonstration listing; not an actionable opportunity")
		if status != models.RiskRed {
			status = models.RiskYellow
		}
	}
	if job.ForexSource == "offline" || job.ForexSource == "frankfurter-stale" {
		flags = append(flags, "Compensation uses offline or stale exchange rates")
		if status != models.RiskRed {
			status = models.RiskYellow
		}
	}
	return status, flags
}

var regexpDataEntry = regexp.MustCompile(`\b(?:data[\s_-]+entry|input[\s_-]+data|entri[\s_-]+data)\b`)
