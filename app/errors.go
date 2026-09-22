package app

type ErrorCode string

const (
	ErrorCodeValidation   ErrorCode = "validation"
	ErrorCodeExecution    ErrorCode = "execution"
	ErrorCodeInspection   ErrorCode = "inspection"
	ErrorCodeConflict     ErrorCode = "conflict"
	ErrorCodeDialog       ErrorCode = "dialog"
	ErrorCodeExport       ErrorCode = "export"
	ErrorCodeConfirmation ErrorCode = "confirmation_required"
)

type ServiceError struct {
	Code    ErrorCode `json:"code"`
	Field   string    `json:"field,omitempty"`
	Message string    `json:"message"`
}

func (e ServiceError) Error() string {
	return e.Message
}

type ValidationResult struct {
	Valid       bool                  `json:"valid"`
	Errors      []ServiceError        `json:"errors"`
	WriteImpact WriteImpactAssessment `json:"writeImpact"`
}

type BenchmarkResult struct {
	RunID  string        `json:"runId,omitempty"`
	State  RunState      `json:"state"`
	Report *Report       `json:"report,omitempty"`
	Error  *ServiceError `json:"error,omitempty"`
}

type CancelResult struct {
	RunID string        `json:"runId,omitempty"`
	State RunState      `json:"state"`
	Error *ServiceError `json:"error,omitempty"`
}

type DirectoryInspectionResult struct {
	Inspection *DirectoryInspection `json:"inspection,omitempty"`
	Error      *ServiceError        `json:"error,omitempty"`
}

type PathSelectionResult struct {
	Path      string        `json:"path,omitempty"`
	Cancelled bool          `json:"cancelled"`
	Suite     *SuiteSummary `json:"suite,omitempty"`
	Error     *ServiceError `json:"error,omitempty"`
}

type ExportResult struct {
	Path      string        `json:"path,omitempty"`
	Cancelled bool          `json:"cancelled"`
	Error     *ServiceError `json:"error,omitempty"`
}
