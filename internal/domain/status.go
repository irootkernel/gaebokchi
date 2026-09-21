package domain

// Status is the terminal outcome of one target query. The string value is
// the internal status/reason; Cell maps it to the WEEKLY USAGE text.
type Status string

const (
	StatusOK Status = "ok"

	StatusNotApplicableMonthly Status = "not_applicable/monthly"
	StatusNotApplicableAPIKey   Status = "not_applicable/api_key"

	StatusUnavailableNoWeekly Status = "unavailable/no_weekly"
	StatusUnavailablePeriod   Status = "unavailable/period"
	StatusUnavailableNoData   Status = "unavailable/no_data"

	StatusAuthRequired  Status = "auth_required"
	StatusUnsupported   Status = "unsupported"
	StatusCLINotFound   Status = "cli_not_found"
	StatusHomeNotFound  Status = "home_not_found"
	StatusHomeError     Status = "home_error"
	StatusTimeout       Status = "timeout"
	StatusProtocolError Status = "protocol_error"
	StatusProviderError Status = "provider_error"
	StatusCancelled     Status = "cancelled"
)

// Cell returns the WEEKLY USAGE column text for a status.
func (s Status) Cell() string {
	switch s {
	case StatusOK:
		return "" // rendered from UsedPercent by the renderer
	case StatusNotApplicableMonthly:
		return "N/A (monthly)"
	case StatusNotApplicableAPIKey:
		return "N/A (API key)"
	case StatusUnavailableNoWeekly:
		return "N/A (no weekly)"
	case StatusUnavailablePeriod:
		return "N/A (period)"
	case StatusUnavailableNoData:
		return "N/A (no data)"
	case StatusAuthRequired:
		return "AUTH REQUIRED"
	case StatusUnsupported:
		return "UNSUPPORTED"
	case StatusCLINotFound:
		return "NOT INSTALLED"
	case StatusHomeNotFound:
		return "HOME NOT FOUND"
	case StatusHomeError:
		return "HOME ERROR"
	case StatusTimeout:
		return "TIMEOUT"
	case StatusProtocolError:
		return "PROTOCOL ERROR"
	case StatusProviderError:
		return "ERROR"
	case StatusCancelled:
		return "CANCELLED"
	default:
		return "ERROR"
	}
}

// ExitClass groups statuses into the exit-code families of ARCH-10.
type ExitClass string

const (
	// ExitClean covers numeric rows and explicit non-applicability.
	ExitClean ExitClass = "clean"
	// ExitIncomplete covers unavailable, failure, and error rows.
	ExitIncomplete ExitClass = "incomplete"
	// ExitSignal covers cancellation by SIGINT/SIGTERM.
	ExitSignal ExitClass = "signal"
)

// ExitClass reports how the status contributes to the process exit code.
func (s Status) ExitClass() ExitClass {
	switch s {
	case StatusOK, StatusNotApplicableMonthly, StatusNotApplicableAPIKey:
		return ExitClean
	case StatusCancelled:
		return ExitSignal
	default:
		return ExitIncomplete
	}
}
