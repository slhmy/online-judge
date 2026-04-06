package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultValidator_Validate(t *testing.T) {
	tests := []struct {
		name     string
		expected []byte
		actual   []byte
		want     Verdict
	}{
		{
			name:     "exact match",
			expected: []byte("1 2 3\n"),
			actual:   []byte("1 2 3\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "match with different trailing whitespace",
			expected: []byte("1 2 3  \n"),
			actual:   []byte("1 2 3\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "match ignoring trailing newlines",
			expected: []byte("1 2 3\n\n"),
			actual:   []byte("1 2 3\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "match with CRLF line endings",
			expected: []byte("1 2 3\r\n"),
			actual:   []byte("1 2 3\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "wrong answer - different values",
			expected: []byte("3\n"),
			actual:   []byte("4\n"),
			want:     VerdictWrongAnswer,
		},
		{
			name:     "wrong answer - extra line",
			expected: []byte("3\n"),
			actual:   []byte("3\n4\n"),
			want:     VerdictWrongAnswer,
		},
		{
			name:     "wrong answer - missing line",
			expected: []byte("3\n4\n"),
			actual:   []byte("3\n"),
			want:     VerdictWrongAnswer,
		},
		{
			name:     "presentation error - different spacing",
			expected: []byte("1 2 3\n"),
			actual:   []byte("1  2  3\n"),
			want:     VerdictPresentation,
		},
		{
			name:     "presentation error - extra newlines",
			expected: []byte("1 2 3\n"),
			actual:   []byte("1 2 3\n\n"),
			want:     VerdictCorrect, // Trailing newlines are ignored
		},
		{
			name:     "multi-line output match",
			expected: []byte("1 2 3\n4 5 6\n"),
			actual:   []byte("1 2 3\n4 5 6\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "multi-line wrong answer",
			expected: []byte("1 2 3\n4 5 6\n"),
			actual:   []byte("1 2 3\n4 5 7\n"),
			want:     VerdictWrongAnswer,
		},
		{
			name:     "empty output match",
			expected: []byte(""),
			actual:   []byte(""),
			want:     VerdictCorrect,
		},
		{
			name:     "expected empty, actual non-empty",
			expected: []byte(""),
			actual:   []byte("1\n"),
			want:     VerdictWrongAnswer,
		},
		{
			name:     "actual empty, expected non-empty",
			expected: []byte("1\n"),
			actual:   []byte(""),
			want:     VerdictWrongAnswer,
		},
		{
			name:     "large numbers",
			expected: []byte("1000000000\n"),
			actual:   []byte("1000000000\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "floating point numbers",
			expected: []byte("3.14159\n"),
			actual:   []byte("3.14159\n"),
			want:     VerdictCorrect,
		},
		{
			name:     "floating point mismatch",
			expected: []byte("3.14159\n"),
			actual:   []byte("3.14160\n"),
			want:     VerdictWrongAnswer,
		},
	}

	validator := NewDefaultValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.expected, tt.actual)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDefaultValidator_WithTabs(t *testing.T) {
	tests := []struct {
		name     string
		expected []byte
		actual   []byte
		want     Verdict
	}{
		{
			name:     "tab and space equivalent",
			expected: []byte("1\t2\n"),
			actual:   []byte("1 2\n"),
			want:     VerdictPresentation,
		},
		{
			name:     "mixed tabs and spaces",
			expected: []byte("1\t\t2\n"),
			actual:   []byte("1  2\n"),
			want:     VerdictPresentation,
		},
	}

	validator := NewDefaultValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.expected, tt.actual)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "LF unchanged",
			input: "line1\nline2\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "CRLF to LF",
			input: "line1\r\nline2\r\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "mixed line endings",
			input: "line1\r\nline2\nline3\r\n",
			want:  "line1\nline2\nline3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeLineEndings(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNormalizeTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no trailing whitespace",
			input: "line1\nline2\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "trailing spaces",
			input: "line1  \nline2\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "trailing tabs",
			input: "line1\t\t\nline2\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "mixed trailing whitespace",
			input: "line1 \t \nline2\n",
			want:  "line1\nline2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTrailingWhitespace(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRemoveExtraWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single spaces preserved",
			input: "a b c",
			want:  "a b c",
		},
		{
			name:  "multiple spaces collapsed",
			input: "a  b   c",
			want:  "a b c",
		},
		{
			name:  "tabs collapsed to spaces",
			input: "a\t\tb",
			want:  "a b",
		},
		{
			name:  "mixed whitespace",
			input: "a \t b \n c",
			want:  "a b c",
		},
		{
			name:  "leading and trailing whitespace",
			input: "  a b  ",
			want:  " a b ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeExtraWhitespace(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsWhitespace(t *testing.T) {
	assert.True(t, isWhitespace(' '))
	assert.True(t, isWhitespace('\t'))
	assert.True(t, isWhitespace('\n'))
	assert.True(t, isWhitespace('\r'))
	assert.False(t, isWhitespace('a'))
	assert.False(t, isWhitespace('1'))
}

func TestValidator_EdgeCases(t *testing.T) {
	validator := NewDefaultValidator()

	t.Run("very long output", func(t *testing.T) {
		expected := make([]byte, 10000)
		actual := make([]byte, 10000)
		for i := range expected {
			expected[i] = 'a'
			actual[i] = 'a'
		}
		result := validator.Validate(expected, actual)
		assert.Equal(t, VerdictCorrect, result)
	})

	t.Run("unicode characters", func(t *testing.T) {
		expected := []byte("Hello 世界\n")
		actual := []byte("Hello 世界\n")
		result := validator.Validate(expected, actual)
		assert.Equal(t, VerdictCorrect, result)
	})

	t.Run("unicode mismatch", func(t *testing.T) {
		expected := []byte("Hello 世界\n")
		actual := []byte("Hello 世界!\n")
		result := validator.Validate(expected, actual)
		assert.Equal(t, VerdictWrongAnswer, result)
	})

	t.Run("null bytes", func(t *testing.T) {
		expected := []byte{0x00, 0x01, 0x02}
		actual := []byte{0x00, 0x01, 0x02}
		result := validator.Validate(expected, actual)
		assert.Equal(t, VerdictCorrect, result)
	})
}

// Benchmark for performance testing
func BenchmarkValidator_Validate(b *testing.B) {
	validator := NewDefaultValidator()
	expected := []byte("1 2 3\n4 5 6\n7 8 9\n")
	actual := []byte("1 2 3\n4 5 6\n7 8 9\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(expected, actual)
	}
}

func BenchmarkValidator_ValidateLarge(b *testing.B) {
	validator := NewDefaultValidator()
	// Create 1MB output
	expected := make([]byte, 1024*1024)
	actual := make([]byte, 1024*1024)
	for i := range expected {
		expected[i] = byte('a' + (i % 26))
		actual[i] = byte('a' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(expected, actual)
	}
}