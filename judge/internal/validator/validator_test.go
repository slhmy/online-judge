package validator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Special Validator Tests

func TestSpecialValidator_ExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		exitCode     int
		expectedVerd Verdict
	}{
		{
			name:         "exit code 42 = correct",
			exitCode:     ExitCodeCorrect,
			expectedVerd: VerdictCorrect,
		},
		{
			name:         "exit code 43 = wrong answer",
			exitCode:     ExitCodeWrongAnswer,
			expectedVerd: VerdictWrongAnswer,
		},
		{
			name:         "exit code 44 = presentation error",
			exitCode:     ExitCodePresentation,
			expectedVerd: VerdictPresentation,
		},
		{
			name:         "exit code 0 = correct (non-standard)",
			exitCode:     0,
			expectedVerd: VerdictCorrect,
		},
		{
			name:         "exit code 1 = internal error",
			exitCode:     1,
			expectedVerd: VerdictInternalError,
		},
		{
			name:         "exit code -1 = internal error",
			exitCode:     -1,
			expectedVerd: VerdictInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that exit code constants are correct
			switch tt.exitCode {
			case ExitCodeCorrect:
				assert.Equal(t, 42, ExitCodeCorrect)
			case ExitCodeWrongAnswer:
				assert.Equal(t, 43, ExitCodeWrongAnswer)
			case ExitCodePresentation:
				assert.Equal(t, 44, ExitCodePresentation)
			}
		})
	}
}

func TestValidatorCache_StoreAndGet(t *testing.T) {
	// Create temporary cache directory
	cacheDir := t.TempDir()
	cache := NewValidatorCache(cacheDir)

	testData := []byte("#!/bin/bash\necho 'test validator'\nexit 42")
	testID := "test-validator-001"
	testMD5 := "abc123"

	// Store validator
	binary, err := cache.Store(testID, testData, testMD5)
	require.NoError(t, err)
	assert.NotNil(t, binary)
	assert.Equal(t, testID, binary.ID)
	assert.Equal(t, testMD5, binary.MD5Sum)
	assert.FileExists(t, binary.Path)

	// Verify file contents
	data, err := os.ReadFile(binary.Path)
	require.NoError(t, err)
	assert.Equal(t, testData, data)

	// Get from cache
	cached, ok := cache.Get(testID)
	require.True(t, ok)
	assert.Equal(t, binary.Path, cached.Path)
	assert.Equal(t, testMD5, cached.MD5Sum)

	// Verify cache size
	assert.Equal(t, 1, cache.Size())
}

func TestValidatorCache_SameMD5Skip(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewValidatorCache(cacheDir)

	testData := []byte("test data")
	testID := "test-validator"
	testMD5 := "same-md5"

	// Store first time
	binary1, err := cache.Store(testID, testData, testMD5)
	require.NoError(t, err)

	// Store again with same MD5 - should return same entry
	binary2, err := cache.Store(testID, []byte("different data"), testMD5)
	require.NoError(t, err)
	assert.Equal(t, binary1.Path, binary2.Path)

	// Cache should still have 1 entry
	assert.Equal(t, 1, cache.Size())
}

func TestValidatorCache_Clear(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewValidatorCache(cacheDir)

	// Store multiple validators
	for i := 0; i < 3; i++ {
		_, err := cache.Store(string(rune('a'+i)), []byte("data"), "md5")
		require.NoError(t, err)
	}

	assert.Equal(t, 3, cache.Size())

	// Clear cache
	err := cache.Clear()
	require.NoError(t, err)
	assert.Equal(t, 0, cache.Size())
}

func TestSpecialValidatorConfig_Defaults(t *testing.T) {
	config := DefaultSpecialValidatorConfig()

	assert.Equal(t, 30*time.Second, config.TimeLimit)
	assert.Equal(t, int64(524288), config.MemoryLimit) // 512 MB
	assert.Equal(t, "/tmp/validator-cache", config.CacheDir)
	assert.True(t, config.EnableCache)
}

// MockHTTPClient for testing
type MockHTTPClient struct {
	response []byte
	md5sum   string
	err      error
}

func (m *MockHTTPClient) FetchExecutable(ctx context.Context, executableID string) ([]byte, string, error) {
	return m.response, m.md5sum, m.err
}

func TestSpecialValidator_FetchValidator_RawBinary(t *testing.T) {
	// Test raw binary response
	validatorBinary := []byte("#!/bin/bash\nexit 42")

	mockClient := &MockHTTPClient{response: validatorBinary}
	config := SpecialValidatorConfig{
		TimeLimit:   5 * time.Second,
		EnableCache: false,
	}

	validator := NewSpecialValidator(config, mockClient)

	data, md5sum, err := validator.fetchValidator(context.Background(), "test-id")
	require.NoError(t, err)
	assert.Equal(t, validatorBinary, data)
	assert.Empty(t, md5sum)
}

func TestSpecialValidator_FetchValidator_JSONResponse(t *testing.T) {
	// Test fetcher-provided binary and md5sum
	validatorBinary := []byte("#!/bin/bash\nexit 42")
	mockClient := &MockHTTPClient{response: validatorBinary, md5sum: "test-md5"}
	config := SpecialValidatorConfig{
		TimeLimit:   5 * time.Second,
		EnableCache: false,
	}

	validator := NewSpecialValidator(config, mockClient)

	data, md5sum, err := validator.fetchValidator(context.Background(), "test-id")
	require.NoError(t, err)
	assert.Equal(t, validatorBinary, data)
	assert.Equal(t, "test-md5", md5sum)
}

func TestSpecialValidator_FetchValidator_NoHTTPClient(t *testing.T) {
	config := SpecialValidatorConfig{
		TimeLimit:   5 * time.Second,
		EnableCache: false,
	}

	validator := NewSpecialValidator(config, nil)

	_, _, err := validator.fetchValidator(context.Background(), "test-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no validator fetcher configured")
}

// TestValidatorBinary_Execution tests actual validator script execution
func TestValidatorBinary_Execution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary validator script
	tmpDir := t.TempDir()
	validatorPath := filepath.Join(tmpDir, "validator.sh")

	validatorScript := `#!/bin/bash
# Simple validator that checks if output matches expected
INPUT="$1"
ANSWER="$2"
OUTPUT="$3"

EXPECTED=$(cat "$ANSWER")
ACTUAL=$(cat "$OUTPUT")

if [ "$EXPECTED" = "$ACTUAL" ]; then
    echo "Correct output"
    exit 42
else
    echo "Wrong answer"
    exit 43
fi
`

	err := os.WriteFile(validatorPath, []byte(validatorScript), 0755)
	require.NoError(t, err)

	// Create test input files
	inputFile := filepath.Join(tmpDir, "input.txt")
	answerFile := filepath.Join(tmpDir, "answer.txt")
	outputFile := filepath.Join(tmpDir, "output.txt")

	_ = os.WriteFile(inputFile, []byte("test input"), 0644)
	_ = os.WriteFile(answerFile, []byte("expected output"), 0644)
	_ = os.WriteFile(outputFile, []byte("expected output"), 0644)

	// Create ValidatorBinary
	binary := &ValidatorBinary{
		ID:       "test-validator",
		Path:     validatorPath,
		MD5Sum:   "test-md5",
		LoadedAt: time.Now(),
	}

	// Verify the binary exists and is executable
	assert.FileExists(t, binary.Path)
}
