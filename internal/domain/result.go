package domain

import "time"

// AccountSource says where a displayed account label came from.
type AccountSource string

const (
	// AccountSourceProtocolEmail is an email read from the target's own
	// protocol session (Codex account/read).
	AccountSourceProtocolEmail AccountSource = "protocol_email"
	// AccountSourceConfiguredAlias is the Codex home basename or the
	// configured Grok alias.
	AccountSourceConfiguredAlias AccountSource = "configured_alias"
)

// PeriodKind describes the subscription window the percentage belongs to.
type PeriodKind string

const (
	PeriodWeekly  PeriodKind = "weekly"
	PeriodMonthly PeriodKind = "monthly"
	PeriodUnknown PeriodKind = "unknown"
)

// Source identifies which provider interface produced the metric.
const (
	SourceCodexAppServer  = "codex_app_server"
	SourceGrokACPBilling  = "grok_acp_billing"
)

// Diagnostic is a safe, application-owned error summary. Codes and messages
// must never embed raw provider payloads or secrets.
type Diagnostic struct {
	Code    string
	Message string
}

// Result is the single terminal outcome of one target query.
type Result struct {
	TargetID      string
	Provider      ProviderKind
	Account       string
	AccountSource AccountSource
	Status        Status
	// UsedPercent is non-nil only for StatusOK. Go zero values must not
	// turn an omitted number into a measured zero.
	UsedPercent *float64
	Period      PeriodKind
	// QuotaID is the provider-local quota identifier (e.g. "codex");
	// it is not a cross-provider unit.
	QuotaID    string
	ResetAt    *time.Time
	ObservedAt time.Time
	Source     string
	Diagnostic Diagnostic
}
