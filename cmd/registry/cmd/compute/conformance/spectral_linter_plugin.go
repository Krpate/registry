// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conformance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/apigee/registry/cmd/registry/core"
	"github.com/apigee/registry/rpc"
)

// SpectralLinter implements the Linter interface, and provides the ability to lint
// API Specs using the spectral linter: https://stoplight.io/open-source/spectral/
type SpectralLinter struct {
	Rules map[string][]string
}

// spectralConfiguration describes a spectral ruleset that is used to lint
// a given API Spec.
type spectralConfiguration struct {
	Extends [][]string      `json:"extends"`
	Rules   map[string]bool `json:"rules"`
}

// spectralLintResult contains metadata related to a rule violation.
type spectralLintResult struct {
	Code     string            `json:"code"`
	Path     []string          `json:"path"`
	Message  string            `json:"message"`
	Severity int32             `json:"severity"`
	Range    spectralLintRange `json:"range"`
	Source   string            `json:"source"`
}

// spectralLintRange is the start and end location for a rule violation.
type spectralLintRange struct {
	Start spectralLintLocation `json:"start"`
	End   spectralLintLocation `json:"end"`
}

// spectralLintLocation is the location in a file for a rule violation.
type spectralLintLocation struct {
	Line      int32 `json:"line"`
	Character int32 `json:"character"`
}

// spectralRunner is an interface through which the Spectral Linter executes.
type spectralRunner interface {
	// Runs the spectral linter with a provided spec and configuration path
	Run(specPath, configPath string) ([]*spectralLintResult, error)
}

// concreteSpectralRunner implements the spectral runner interface.
type concreteSpectralRunner struct{}

func NewSpectralLinter() SpectralLinter {
	return SpectralLinter{
		Rules: make(map[string][]string),
	}
}

// AddRule registers a specific linter rule onto a given mime type.
func (linter SpectralLinter) AddRule(mimeType string, rule string) error {
	// Check if the linter supports the mime type.
	if !linter.SupportsMimeType(mimeType) {
		return createUnsupportedMimeTypeError(mimeType)
	}

	linter.Rules[mimeType] = append(linter.Rules[mimeType], rule)
	return nil
}

// GetName returns the spectral, the name of this linter.
func (linter SpectralLinter) GetName() string {
	return "spectral"
}

// SupportsMimeType returns the mime types that are supported by Spectral's core.
// rulesets. Currently, Spectral supports OpenAPI v2/v3 and AsyncAPI v2:
// https://meta.stoplight.io/docs/spectral/ZG9jOjYyMDc0NA-rulesets#core-rulesets
func (linter SpectralLinter) SupportsMimeType(mimeType string) bool {
	return core.IsOpenAPIv2(mimeType) ||
		core.IsOpenAPIv3(mimeType) ||
		core.IsAsyncAPIv2(mimeType)
}

// LintSpec lints the spec pointed at by a spec path, which has a provided mime type.
// It returns the results as a list of LintProblem objects.
func (linter SpectralLinter) LintSpec(
	mimeType string,
	specPath string,
) ([]*rpc.LintProblem, error) {
	return linter.LintSpecImpl(mimeType, specPath, &concreteSpectralRunner{})
}

func (linter SpectralLinter) LintSpecImpl(
	mimeType string,
	specPath string,
	runner spectralRunner,
) ([]*rpc.LintProblem, error) {
	// Check if the linter supports the mime type
	if !linter.SupportsMimeType(mimeType) {
		return nil, createUnsupportedMimeTypeError(mimeType)
	}

	// Create a temporary directory to store the configuration.
	root, err := createTemporaryConfigDirectory("spectral-config-")
	if err != nil {
		return nil, err
	}

	// Defer the deletion of the the temporary directory.
	defer os.RemoveAll(root)

	// Create configuration file for Spectral to execute the correct rules
	configPath, err := linter.createConfigurationFile(root, mimeType)
	if err != nil {
		return nil, err
	}

	// Execute the spectral linter.
	lintResults, err := runner.Run(specPath, configPath)
	if err != nil {
		return nil, err
	}

	// Get the lint results as a LintFile object from the spectral output file
	lintProblems, err := getLintProblemsFromSpectralResults(lintResults)
	if err != nil {
		return nil, err
	}

	return lintProblems, nil
}

// Creates a configuration file and returns its path.
func (linter SpectralLinter) createConfigurationFile(root string, mimeType string) (string, error) {
	// Create the spectral configuration.
	configuration := spectralConfiguration{}
	configuration.Rules = make(map[string]bool)
	if core.IsOpenAPIv2(mimeType) || core.IsOpenAPIv3(mimeType) {
		configuration.Extends = [][]string{{"spectral:oas", "off"}}
	} else {
		configuration.Extends = [][]string{{"spectral:asyncapi", "off"}}
	}
	for _, rules := range linter.Rules {
		for _, ruleName := range rules {
			configuration.Rules[ruleName] = true
		}
	}

	// Marshal the configuration into a file.
	file, err := json.MarshalIndent(configuration, "", " ")
	if err != nil {
		return "", err
	}

	// Write the configuration to the temporary directory.
	configPath := filepath.Join(root, "spectral.json")
	err = writeBytesToFile(configPath, file)
	if err != nil {
		return "", err
	}

	return configPath, nil
}

func writeBytesToFile(filepath string, file []byte) error {
	return ioutil.WriteFile(filepath, file, 0644)
}

func createTemporaryConfigDirectory(prefix string) (string, error) {
	return ioutil.TempDir("", prefix)
}

func getLintProblemsFromSpectralResults(
	lintResults []*spectralLintResult,
) ([]*rpc.LintProblem, error) {
	problems := make([]*rpc.LintProblem, len(lintResults))
	for i, result := range lintResults {
		problem := &rpc.LintProblem{
			Message:    result.Message,
			RuleId:     result.Code,
			RuleDocUri: "https://meta.stoplight.io/docs/spectral/docs/reference/openapi-rules.md#" + result.Code,
			Location: &rpc.LintLocation{
				StartPosition: &rpc.LintPosition{
					LineNumber:   result.Range.Start.Line + 1,
					ColumnNumber: result.Range.Start.Character + 1,
				},
				EndPosition: &rpc.LintPosition{
					LineNumber:   result.Range.End.Line + 1,
					ColumnNumber: result.Range.End.Character,
				},
			},
		}
		problems[i] = problem
	}
	return problems, nil
}

// createUnsupportedMimeTypeError returns an error for unsupported mime types.
func createUnsupportedMimeTypeError(mimeType string) error {
	return fmt.Errorf("mime type %s is not supported by the spectral linter", mimeType)
}

func (*concreteSpectralRunner) Run(
	specPath,
	configPath string,
) ([]*spectralLintResult, error) {
	// Create a temporary destination directory to store the output.
	root, err := createTemporaryConfigDirectory("spectral-output-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)

	// Set the destination path of the spectral output.
	outputPath := filepath.Join(root, "spectral-lint.json")

	cmd := exec.Command("spectral",
		"lint", specPath,
		"--r", configPath,
		"--f", "json",
		"--output", outputPath,
	)

	// Ignore errors from Spectral because Spectral returns an
	// error result when APIs have errors.
	_ = cmd.Run()

	// Read and parse the spectral output.
	b, err := ioutil.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}
	var lintResults []*spectralLintResult
	err = json.Unmarshal(b, &lintResults)
	if err != nil {
		return nil, err
	}

	return lintResults, nil
}
