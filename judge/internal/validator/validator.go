package validator

import (
	"bytes"
	"context"
	"strings"
)

type Verdict string

const (
	VerdictCorrect       Verdict = "correct"
	VerdictWrongAnswer   Verdict = "wrong-answer"
	VerdictPresentation  Verdict = "presentation"
	VerdictInternalError Verdict = "internal-error"
)

// Validator interface for output validation
type Validator interface {
	// Validate compares expected and actual output
	Validate(expected, actual []byte) Verdict
}

// SpecialValidatorInterface for validators that need full context
type SpecialValidatorInterface interface {
	// ValidateWithInput validates with full context (input, expected, actual)
	ValidateWithInput(ctx context.Context, validatorID string, args string, input, expected, actual []byte) (Verdict, string)
}

// DefaultValidator implements standard output comparison
type DefaultValidator struct {
	IgnoreTrailingWhitespace bool
	IgnoreCase               bool
}

func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		IgnoreTrailingWhitespace: true,
		IgnoreCase:               false,
	}
}

func (v *DefaultValidator) Validate(expected, actual []byte) Verdict {
	// Normalize line endings
	expectedStr := normalizeLineEndings(string(expected))
	actualStr := normalizeLineEndings(string(actual))

	if v.IgnoreTrailingWhitespace {
		expectedStr = normalizeTrailingWhitespace(expectedStr)
		actualStr = normalizeTrailingWhitespace(actualStr)
		// Also trim trailing newlines
		expectedStr = strings.TrimRight(expectedStr, "\n")
		actualStr = strings.TrimRight(actualStr, "\n")
	}

	if v.IgnoreCase {
		expectedStr = strings.ToLower(expectedStr)
		actualStr = strings.ToLower(actualStr)
	}

	// Exact match
	if expectedStr == actualStr {
		return VerdictCorrect
	}

	// Check for presentation error
	if v.isPresentationError(expectedStr, actualStr) {
		return VerdictPresentation
	}

	return VerdictWrongAnswer
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func normalizeTrailingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func (v *DefaultValidator) isPresentationError(expected, actual string) bool {
	// Remove all whitespace and compare
	expectedNorm := removeExtraWhitespace(expected)
	actualNorm := removeExtraWhitespace(actual)

	return expectedNorm == actualNorm
}

func removeExtraWhitespace(s string) string {
	var buf bytes.Buffer
	inSpace := false

	for _, r := range s {
		if isWhitespace(r) {
			if !inSpace {
				buf.WriteRune(' ')
				inSpace = true
			}
		} else {
			buf.WriteRune(r)
			inSpace = false
		}
	}

	return buf.String()
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}