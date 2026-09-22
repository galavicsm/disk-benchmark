package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"diskbenchmark/benchmark"
	"diskbenchmark/output"
)

type ExportFormat string

const (
	ExportJSON ExportFormat = "json"
	ExportCSV  ExportFormat = "csv"
)

type ExportRequest struct {
	RunID  string       `json:"runId"`
	Format ExportFormat `json:"format"`
}

func (s *Service) ExportReport(request ExportRequest) ExportResult {
	format, err := output.ParseFormat(strings.ToLower(string(request.Format)))
	if err != nil || (format != output.JSON && format != output.CSV) {
		if err == nil {
			err = fmt.Errorf("unsupported report format %q; use json or csv", request.Format)
		}
		return exportError(err)
	}
	if request.RunID == "" {
		return exportError(fmt.Errorf("report run ID must not be empty"))
	}

	s.mu.Lock()
	report, ok := s.reports[request.RunID]
	s.mu.Unlock()
	if !ok {
		return exportError(fmt.Errorf("no completed report exists for run %q", request.RunID))
	}
	if s.dialogs.SelectSaveFile == nil {
		return exportError(fmt.Errorf("native save dialog is unavailable"))
	}

	extension := string(format)
	path, err := s.dialogs.SelectSaveFile(SaveDialogRequest{
		Title:           "Export benchmark report",
		DefaultFilename: "diskbenchmark-report." + extension,
		DisplayName:     strings.ToUpper(extension) + " report (*." + extension + ")",
		Pattern:         "*." + extension,
	})
	if err != nil {
		return exportError(fmt.Errorf("select report destination: %w", err))
	}
	if path == "" {
		return ExportResult{Cancelled: true}
	}
	if filepath.Ext(path) == "" {
		path += "." + extension
	}
	if err := writeReport(path, format, report); err != nil {
		return exportError(err)
	}
	return ExportResult{Path: path}
}

func writeReport(path string, format output.Format, report benchmark.Report) (err error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary report file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		removeErr := os.Remove(temporaryPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove temporary report file: %w", removeErr))
		}
	}()

	if renderErr := output.Render(temporary, format, report); renderErr != nil {
		closeErr := temporary.Close()
		return errors.Join(
			fmt.Errorf("render report: %w", renderErr),
			wrapCloseReportError(closeErr),
		)
	}
	if syncErr := temporary.Sync(); syncErr != nil {
		closeErr := temporary.Close()
		return errors.Join(
			fmt.Errorf("flush report: %w", syncErr),
			wrapCloseReportError(closeErr),
		)
	}
	if closeErr := temporary.Close(); closeErr != nil {
		return fmt.Errorf("close report: %w", closeErr)
	}
	if renameErr := os.Rename(temporaryPath, path); renameErr != nil {
		return fmt.Errorf("replace report %q: %w", path, renameErr)
	}
	return nil
}

func wrapCloseReportError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("close report: %w", err)
}

func exportError(err error) ExportResult {
	return ExportResult{Error: serviceError(ErrorCodeExport, "", err)}
}
