package game

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateGuess(t *testing.T) {
	// Valid inputs
	guess, err := ValidateGuess("2000")
	assert.NoError(t, err)
	assert.Equal(t, 2000, guess)

	guess, err = ValidateGuess(" 9931")
	assert.NoError(t, err)
	assert.Equal(t, 9931, guess)

	// Too short / too long
	_, err = ValidateGuess("123")
	assert.Error(t, err)

	_, err = ValidateGuess("12345")
	assert.Error(t, err)

	// Non-digit characters
	_, err = ValidateGuess("12a4")
	assert.Error(t, err)

	_, err = ValidateGuess("$123")
	assert.Error(t, err)

	// Whitespace only / empty
	_, err = ValidateGuess("    ")
	assert.Error(t, err)

	_, err = ValidateGuess("")
	assert.Error(t, err)
}


func TestGenerateSecretCodeRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := GenerateSecretCode(Medium)
		assert.GreaterOrEqual(t, code, 1000, "code should be >= 1000")
		assert.LessOrEqual(t, code, 9999, "code should be <= 9999")
	}
}

func TestGenerateSecretCodeEasyNoRepeats(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := GenerateSecretCode(Easy)
		s := strconv.Itoa(code)
		seen := make(map[rune]bool)
		for _, c := range s {
			assert.False(t, seen[c], "Easy code %d has repeated digit %c", code, c)
			seen[c] = true
		}
	}
}

func TestGenerateSecretCodeHardConstraints(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := GenerateSecretCode(Hard)
		s := strconv.Itoa(code)

		// must have at least one repeating digit
		seen := make(map[rune]bool)
		hasRepeat := false
		for _, c := range s {
			if seen[c] {
				hasRepeat = true
			}
			seen[c] = true
		}
		assert.True(t, hasRepeat, "Hard code %d should have a repeating digit", code)

		// sum of digits must be prime
		sum := 0
		for _, c := range s {
			sum += int(c - '0')
		}
		assert.True(t, isPrime(sum), "Hard code %d has digit sum %d which is not prime", code, sum)
	}
}

func TestGenerateSecretCodePalindromeReplacedWith7777(t *testing.T) {
	// A number whose adjustment produces a palindrome must become 7777.
	// 1001 → sum=2 (even) → reverse → "1001" (palindrome) → 7777
	code := applySecretCodeLogic(1001)
	assert.Equal(t, 7777, code)
}

func TestGenerateSecretCodeEvenSumReversed(t *testing.T) {
	// 1234 → sum=10 (even) → reverse → 4321 (not a palindrome)
	code := applySecretCodeLogic(1234)
	assert.Equal(t, 4321, code)
}

func TestGenerateSecretCodeOddSumIncremented(t *testing.T) {
	// 1235 → sum=11 (odd) → increment each digit → 2346
	code := applySecretCodeLogic(1235)
	assert.Equal(t, 2346, code)
}

func TestGenerateSecretCodeOddSumWrapAround(t *testing.T) {
	// 1299 → sum=21 (odd) → increment each digit, 9→0 → 2300
	code := applySecretCodeLogic(1299)
	assert.Equal(t, 2300, code)
}

func TestGenerateFeedbackCorrect(t *testing.T) {
	result := GenerateFeedback(1234, 1234)
	assert.Equal(t, "Congratulations! You guessed the correct number!", result)
}

func TestGenerateFeedbackCountsAndHints(t *testing.T) {
	// secret=1234, guess=1243: digits 1,2 correct in place; 3,4 misplaced
	result := GenerateFeedback(1234, 1243)
	assert.Contains(t, result, "Correct digits in right position: 2")
	assert.Contains(t, result, "Correct digits in wrong position: 2")
	assert.Contains(t, result, "first half") // 1,2 are correct in first half
}

func TestGenerateFeedbackNoMatches(t *testing.T) {
	// secret=1234, guess=5678: no matches
	result := GenerateFeedback(1234, 5678)
	assert.Contains(t, result, "Correct digits in right position: 0")
	assert.Contains(t, result, "Correct digits in wrong position: 0")
}

func TestGenerateFeedbackSecondHalfCorrect(t *testing.T) {
	// secret=1234, guess=5634: only digits at index 2,3 correct (second half)
	result := GenerateFeedback(1234, 5634)
	assert.Contains(t, result, "Correct digits in right position: 2")
	assert.Contains(t, result, "second half")
}

func TestGenerateTimestampPrefixFormat(t *testing.T) {
	before := time.Now().Unix()
	prefix := GenerateTimestampPrefix()
	after := time.Now().Unix()

	// Must start with "TIME: "
	assert.True(t, strings.HasPrefix(prefix, "TIME: "), "prefix should start with 'TIME: ', got: %s", prefix)

	// The numeric part must be a valid unix timestamp within the test window
	timestampStr := strings.TrimPrefix(prefix, "TIME: ")
	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	assert.NoError(t, err, "timestamp part should be a valid integer, got: %s", timestampStr)
	assert.GreaterOrEqual(t, ts, before, "timestamp should be >= time before call")
	assert.LessOrEqual(t, ts, after, "timestamp should be <= time after call")
}

func applySecretCodeLogic(n int) int {
	result, _ := applyRules(n)
	return result
}

// TestGenerateSecretCodeDeterministic overrides the package-level rng with a
// seeded source so GenerateSecretCode produces a known, repeatable result.
// seed=42 → raw=1305, sum=9 (odd) → increment each digit → 2416
func TestGenerateSecretCodeDeterministic(t *testing.T) {
	rng = rand.New(rand.NewSource(42))
	code := GenerateSecretCode(Medium)
	assert.Equal(t, 2416, code)
}
