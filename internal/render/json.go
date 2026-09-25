package render

import (
	"bytes"
	"encoding/json"

	"gaebokchi/internal/domain"
)

// JSON projects the ordered snapshot into the public machine-readable shape.
// It does not serialize Result directly: provider provenance, raw diagnostics,
// and other internal fields are outside the output contract.
func JSON(results []domain.Result) (string, error) {
	type row struct {
		TargetID      string              `json:"target_id"`
		Provider      domain.ProviderKind `json:"provider"`
		Account       string              `json:"account"`
		Status        domain.Status       `json:"status"`
		UsedPercent   *float64            `json:"used_percent"`
		CleanupFailed bool                `json:"cleanup_failed"`
	}
	type snapshot struct {
		Results []row `json:"results"`
	}

	rows := make([]row, len(results))
	for i, result := range results {
		rows[i] = row{
			TargetID:      result.TargetID,
			Provider:      result.Provider,
			Account:       SanitizeLabel(result.Account),
			Status:        result.Status,
			UsedPercent:   result.UsedPercent,
			CleanupFailed: result.CleanupFailed,
		}
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(snapshot{Results: rows}); err != nil {
		return "", err
	}
	return buf.String(), nil
}
