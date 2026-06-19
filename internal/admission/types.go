package admission

// FailureTaxonomy provides minimal classification for withheld/failed/blocked outcomes.
type FailureTaxonomy struct {
	FailureClass   string `json:"failure_class"`
	FailureStage   string `json:"failure_stage"`
	Recoverability string `json:"recoverability"`
	NextAction     string `json:"next_action"`
}

type AdmissionOptions struct {
	Strict              bool
	ContextFloor        bool
	RequireDeepApproval bool
	RequireEvidenceHash bool
}

// Result is the output of the admission decision engine.
type Result struct {
	Outcome           string           `json:"outcome"`
	AcceptanceStatus  string           `json:"acceptance_status"`
	Errors            []string         `json:"errors"`
	Notes             []string         `json:"notes"`
	BlockingPredicate string           `json:"blocking_predicate,omitempty"`
	WithheldReason    *FailureTaxonomy `json:"withheld_reason,omitempty"`
}

type evidenceFinding struct {
	message   string
	predicate string
}

type evidenceResult struct {
	errors []evidenceFinding
	notes  []string
}
