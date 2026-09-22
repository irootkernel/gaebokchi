package codex

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gaebokchi/internal/domain"
)

// weeklyWindowMins is the provider's seven-day weekly window duration;
// selection is by this value, never by slot name.
const weeklyWindowMins = 10080

// quotaOutcome is the pure normalization result of one rateLimits/read
// payload.
type quotaOutcome struct {
	Status  domain.Status
	Percent *float64
	ResetAt *time.Time
	QuotaID string
}

func outcome(s domain.Status) quotaOutcome { return quotaOutcome{Status: s} }

// wire types preserve presence: pointers and raw messages keep omitted
// fields distinguishable from zero values.

type rateLimitsResultWire struct {
	RateLimitsByLimitID json.RawMessage  `json:"rateLimitsByLimitId"`
	RateLimits          *json.RawMessage `json:"rateLimits"`
}

type bucketWire struct {
	LimitID   *string          `json:"limitId"`
	Primary   *json.RawMessage `json:"primary"`
	Secondary *json.RawMessage `json:"secondary"`
}

// normalizeQuota applies the deterministic selection policy of ARCH-07.
// The input is the raw result object of account/rateLimits/read.
func normalizeQuota(raw json.RawMessage) quotaOutcome {
	if len(raw) == 0 || string(raw) == "null" {
		return outcome(domain.StatusUnavailableNoData)
	}
	var outer rateLimitsResultWire
	if err := json.Unmarshal(raw, &outer); err != nil {
		return outcome(domain.StatusProtocolError)
	}

	bucket, status, ok := selectBucket(outer)
	if !ok {
		return outcome(status)
	}
	return selectWeeklyWindow(bucket)
}

// selectBucket picks the authoritative bucket: the non-empty
// rateLimitsByLimitId["codex"] entry, or the compatible legacy object
// otherwise. A non-empty map without a codex entry never falls through
// to other buckets or legacy data.
func selectBucket(outer rateLimitsResultWire) (*bucketWire, domain.Status, bool) {
	if len(outer.RateLimitsByLimitID) > 0 && string(outer.RateLimitsByLimitID) != "null" {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(outer.RateLimitsByLimitID, &m); err != nil {
			return nil, domain.StatusProtocolError, false
		}
		if len(m) == 0 {
			// Treat an explicitly empty map as absent: legacy path.
			return legacyBucket(outer)
		}
		entry, present := m["codex"]
		if !present {
			return nil, domain.StatusUnavailableNoWeekly, false
		}
		if string(entry) == "null" {
			return nil, domain.StatusUnavailableNoData, false
		}
		var b bucketWire
		if err := json.Unmarshal(entry, &b); err != nil {
			return nil, domain.StatusProtocolError, false
		}
		return &b, domain.StatusOK, true
	}
	return legacyBucket(outer)
}

// legacyBucket accepts an explicit limitId=="codex". Absent or null
// limitId has no captured fixture and is therefore not eligible.
func legacyBucket(outer rateLimitsResultWire) (*bucketWire, domain.Status, bool) {
	if outer.RateLimits == nil || string(*outer.RateLimits) == "null" {
		return nil, domain.StatusUnavailableNoWeekly, false
	}
	var b bucketWire
	if err := json.Unmarshal(*outer.RateLimits, &b); err != nil {
		return nil, domain.StatusProtocolError, false
	}
	if b.LimitID == nil || *b.LimitID == "" {
		return nil, domain.StatusUnavailableNoWeekly, false
	}
	if *b.LimitID != "codex" {
		return nil, domain.StatusUnavailableNoWeekly, false
	}
	return &b, domain.StatusOK, true
}

// windowWire decodes one rate-limit window. Fields stay raw so numeric
// types can be distinguished from numeric strings and explicit nulls.
type windowWire struct {
	UsedPercent        *json.RawMessage `json:"usedPercent"`
	WindowDurationMins *json.RawMessage `json:"windowDurationMins"`
	ResetsAt           *json.RawMessage `json:"resetsAt"`
}

// isJSONNumber reports whether raw is a JSON number literal (not a
// string, bool, object, or null).
func isJSONNumber(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return false
	}
	c := s[0]
	return c == '-' || c == '+' || (c >= '0' && c <= '9')
}

// weeklyWindow is a validated weekly slot.
type weeklyWindow struct {
	percent float64
	resetAt *time.Time
}

// selectWeeklyWindow implements steps 3-6 of the policy: inspect
// primary and secondary, match windowDurationMins == 10080 exactly,
// require a valid percentage, and treat exactly-agreeing duplicates as
// one window.
func selectWeeklyWindow(b *bucketWire) quotaOutcome {
	var candidates []weeklyWindow
	for _, raw := range []*json.RawMessage{b.Primary, b.Secondary} {
		if raw == nil || string(*raw) == "null" {
			continue // null or absent windows are allowed
		}
		var w windowWire
		if err := json.Unmarshal(*raw, &w); err != nil {
			return outcome(domain.StatusProtocolError)
		}
		if w.WindowDurationMins == nil || string(*w.WindowDurationMins) == "null" {
			continue // no duration: cannot establish a weekly window
		}
		if !isJSONNumber(*w.WindowDurationMins) {
			return outcome(domain.StatusProtocolError)
		}
		dur, ok := integerValue(json.Number(string(*w.WindowDurationMins)))
		if !ok {
			return outcome(domain.StatusProtocolError)
		}
		if dur != weeklyWindowMins {
			continue
		}
		// This slot is weekly: its usage must be a valid number.
		if w.UsedPercent == nil || string(*w.UsedPercent) == "null" {
			return outcome(domain.StatusUnavailableNoData)
		}
		if !isJSONNumber(*w.UsedPercent) {
			return outcome(domain.StatusProtocolError)
		}
		var percent float64
		if err := json.Unmarshal(*w.UsedPercent, &percent); err != nil {
			return outcome(domain.StatusProtocolError)
		}
		if !domain.ValidatePercent(percent) {
			return outcome(domain.StatusProtocolError)
		}
		candidates = append(candidates, weeklyWindow{
			percent: percent,
			resetAt: optionalResetTime(w.ResetsAt),
		})
	}

	switch len(candidates) {
	case 0:
		return outcome(domain.StatusUnavailableNoWeekly)
	case 1:
		c := candidates[0]
		return quotaOutcome{Status: domain.StatusOK, Percent: &c.percent, ResetAt: c.resetAt, QuotaID: "codex"}
	default:
		// Both windows are weekly: they must agree exactly to count as
		// a duplicate representation of one window.
		a, b2 := candidates[0], candidates[1]
		if a.percent != b2.percent || !resetsAgree(a.resetAt, b2.resetAt) {
			return outcome(domain.StatusProtocolError)
		}
		return quotaOutcome{Status: domain.StatusOK, Percent: &a.percent, ResetAt: a.resetAt, QuotaID: "codex"}
	}
}

// integerValue reports whether n is a JSON number with an integer value.
func integerValue(n json.Number) (int64, bool) {
	s := n.String()
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, true
	}
	// Reject non-integer numerics such as 10080.5 or 1e4.
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return 0, false
	}
	return 0, false
}

// optionalResetTime converts an optional resetsAt; malformed optional
// reset metadata is discarded rather than poisoning the percentage.
func optionalResetTime(raw *json.RawMessage) *time.Time {
	if raw == nil || string(*raw) == "null" {
		return nil
	}
	if !isJSONNumber(*raw) {
		return nil // malformed optional reset: discard
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(*raw)), 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(v, 0).UTC()
	return &t
}

func resetsAgree(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

// formatQuotaError renders a safe, fixed diagnostic for a quota status.
func formatQuotaError(status domain.Status, targetID string) string {
	switch status {
	case domain.StatusProtocolError:
		return fmt.Sprintf("%s: rate-limit response did not match the qualified shape", targetID)
	case domain.StatusUnavailableNoWeekly:
		return fmt.Sprintf("%s: no weekly window in the codex quota", targetID)
	case domain.StatusUnavailableNoData:
		return fmt.Sprintf("%s: weekly window present but usage missing", targetID)
	default:
		return fmt.Sprintf("%s: quota unavailable", targetID)
	}
}
