package app

import (
	"fmt"

	"diskbenchmark/suite"
)

type SaveDialogRequest struct {
	Title           string
	DefaultFilename string
	DisplayName     string
	Pattern         string
}

type DialogProvider struct {
	SelectDirectory func() (string, error)
	SelectSuiteFile func() (string, error)
	SelectSaveFile  func(SaveDialogRequest) (string, error)
}

func WithDialogProvider(provider DialogProvider) ServiceOption {
	return func(service *Service) {
		service.dialogs = provider
	}
}

func (s *Service) SelectDirectory() PathSelectionResult {
	if s.dialogs.SelectDirectory == nil {
		return pathSelectionError(fmt.Errorf("native directory dialog is unavailable"))
	}
	path, err := s.dialogs.SelectDirectory()
	if err != nil {
		return pathSelectionError(fmt.Errorf("select benchmark directory: %w", err))
	}
	return PathSelectionResult{Path: path, Cancelled: path == ""}
}

func (s *Service) SelectSuiteFile() PathSelectionResult {
	if s.dialogs.SelectSuiteFile == nil {
		return pathSelectionError(fmt.Errorf("native suite-file dialog is unavailable"))
	}
	path, err := s.dialogs.SelectSuiteFile()
	if err != nil {
		return pathSelectionError(fmt.Errorf("select workload suite: %w", err))
	}
	if path == "" {
		return PathSelectionResult{Cancelled: true}
	}
	file, err := suite.LoadFile(path)
	if err != nil {
		return PathSelectionResult{
			Path:  path,
			Error: serviceError(ErrorCodeValidation, "suiteFile", err),
		}
	}
	summary := newSuiteSummary(file)
	return PathSelectionResult{Path: path, Suite: &summary}
}

func pathSelectionError(err error) PathSelectionResult {
	return PathSelectionResult{Error: serviceError(ErrorCodeDialog, "", err)}
}
