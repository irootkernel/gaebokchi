package grok

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gaebokchi/internal/domain"
)

// Qualified period enum values from the TASK-001 capture.
const (
	periodWeekly  = "USAGE_PERIOD_TYPE_WEEKLY"
	periodMonthly = "USAGE_PERIOD_TYPE_MONTHLY"
)

// billingOutcome is the pure normalization result of one billing
// response payload.
type billingOutcome struct {
	Status  domain.Status
	Percent *float64
	Period  domain.PeriodKind
	ResetAt *time.Time
}

func outcome(s domain.Status) billingOutcome { return billingOutcome{Status: s} }

// billingWire preserves presence of the qualified members.
type billingWire struct {
	Config *json.RawMessage `json:"config"`
}

type configWire struct {
	CurrentPeriod     *json.RawMessage `json:"currentPeriod"`
	CreditUsagePercent *json.RawMessage `json:"creditUsagePercent"`
}

type periodWire struct {
	Type *json.RawMessage `json:"type"`
	End  *json.RawMessage `json:"end"`
}

// isJSONNumber reports whether raw is a JSON number literal.
func isJSONNumber(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return false
	}
	c := s[0]
	return c == '-' || c == '+' || (c >= '0' && c <= '9')
}

// normalizeBilling maps a billing result payload per the ARCH-08 table.
// The input is the raw result of `_x.ai/billing`.
func normalizeBilling(raw json.RawMessage) billingOutcome {
	if len(raw) == 0 {
		return outcome(domain.StatusProtocolError)
	}
	if string(raw) == "null" {
		// A null result carries no config: data is absent.
		return outcome(domain.StatusUnavailableNoData)
	}
	var bw billingWire
	if err := json.Unmarshal(raw, &bw); err != nil {
		return outcome(domain.StatusProtocolError)
	}
	if bw.Config == nil || string(*bw.Config) == "null" {
		return outcome(domain.StatusUnavailableNoData)
	}
	var cfg configWire
	if err := json.Unmarshal(*bw.Config, &cfg); err != nil {
		return outcome(domain.StatusProtocolError)
	}

	if cfg.CurrentPeriod == nil || string(*cfg.CurrentPeriod) == "null" {
		// Absent period metadata — including legacy monthly-only
		// payloads — never establishes a week.
		return outcome(domain.StatusUnavailablePeriod)
	}
	var period periodWire
	if err := json.Unmarshal(*cfg.CurrentPeriod, &period); err != nil {
		return outcome(domain.StatusProtocolError)
	}
	if period.Type == nil || string(*period.Type) == "null" {
		return outcome(domain.StatusUnavailablePeriod)
	}
	var periodType string
	if err := json.Unmarshal(*period.Type, &periodType); err != nil {
		return outcome(domain.StatusProtocolError)
	}

	switch periodType {
	case periodMonthly:
		// An explicit monthly enum is truthful non-applicability; its
		// percentage must never occupy the weekly cell. The established
		// period kind and the optional period end stay in the result as
		// provenance.
		return billingOutcome{
			Status:  domain.StatusNotApplicableMonthly,
			Period:  domain.PeriodMonthly,
			ResetAt: optionalRFC3339(period.End),
		}
	case periodWeekly:
		return weeklyResult(cfg, period)
	default:
		// Unknown well-formed enum.
		return outcome(domain.StatusUnavailablePeriod)
	}
}

func weeklyResult(cfg configWire, period periodWire) billingOutcome {
	// The weekly enum was established before any percentage was read,
	// so period provenance survives even when usage is missing or
	// malformed.
	established := billingOutcome{Period: domain.PeriodWeekly, ResetAt: optionalRFC3339(period.End)}
	if cfg.CreditUsagePercent == nil || string(*cfg.CreditUsagePercent) == "null" {
		established.Status = domain.StatusUnavailableNoData
		return established
	}
	if !isJSONNumber(*cfg.CreditUsagePercent) {
		established.Status = domain.StatusProtocolError
		return established
	}
	var percent float64
	if err := json.Unmarshal(*cfg.CreditUsagePercent, &percent); err != nil {
		established.Status = domain.StatusProtocolError
		return established
	}
	if !domain.ValidatePercent(percent) {
		established.Status = domain.StatusProtocolError
		return established
	}
	established.Status = domain.StatusOK
	established.Percent = &percent
	return established
}

// optionalRFC3339 parses an optional period end; a malformed optional
// end is discarded without replacing the verified percentage.
func optionalRFC3339(raw *json.RawMessage) *time.Time {
	if raw == nil || string(*raw) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(*raw, &s); err != nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// formatBillingError renders a safe fixed diagnostic.
func formatBillingError(status domain.Status, targetID string) string {
	switch status {
	case domain.StatusNotApplicableMonthly:
		return fmt.Sprintf("%s: grok account reports a monthly period, not weekly", targetID)
	case domain.StatusUnavailablePeriod:
		return fmt.Sprintf("%s: billing response does not establish a weekly period", targetID)
	case domain.StatusUnavailableNoData:
		return fmt.Sprintf("%s: weekly period present but usage missing", targetID)
	case domain.StatusProtocolError:
		return fmt.Sprintf("%s: billing response did not match the qualified shape", targetID)
	case domain.StatusUnsupported:
		return fmt.Sprintf("%s: billing extension not available in the installed grok CLI", targetID)
	default:
		return fmt.Sprintf("%s: billing unavailable", targetID)
	}
}
