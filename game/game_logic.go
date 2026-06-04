package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Difficulty int

const (
	Easy   Difficulty = iota
	Medium Difficulty = iota
	Hard   Difficulty = iota
)

// rng is the random source used by GenerateSecretCode.
// Tests can replace this with a seeded source for deterministic output.
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
var rngMu sync.Mutex

func ValidateGuess(input string) (int, error) {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) != 4 {
		return 0, fmt.Errorf("input must be exactly 4 digits, got %d characters", len(trimmed))
	}
	for _, c := range trimmed {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("input must contain only digits")
		}
	}
	guess, _ := strconv.Atoi(trimmed)
	return guess, nil
}

func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func sumDigits(s string) int {
	sum := 0
	for _, c := range s {
		sum += int(c - '0')
	}
	return sum
}

func hasRepeatingDigit(s string) bool {
	var seen [10]bool
	for _, c := range s {
		d := c - '0'
		if seen[d] {
			return true
		}
		seen[d] = true
	}
	return false
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func applyRules(n int) (int, bool) {
	s := strconv.Itoa(n)

	var adjusted string
	if sumDigits(s)%2 == 0 {
		adjusted = reverseString(s)
	} else {
		runes := []rune(s)
		for i, c := range runes {
			runes[i] = '0' + (c-'0'+1)%10
		}
		adjusted = string(runes)
	}

	result, _ := strconv.Atoi(adjusted)
	if result < 1000 {
		return 0, false // leading zero, re-generate
	}

	if adjusted == reverseString(adjusted) {
		adjusted = "7777"
	}

	result, _ = strconv.Atoi(adjusted)
	return result, true
}

func GenerateSecretCode(difficulty Difficulty) int {
	for {
		rngMu.Lock()
		n := rng.Intn(9000) + 1000 // random 1000–9999
		rngMu.Unlock()

		switch difficulty {
		case Easy:
			// Apply the same rules as Medium, then reject if result has repeated digits
			result, ok := applyRules(n)
			if !ok {
				continue
			}
			if hasRepeatingDigit(strconv.Itoa(result)) {
				continue
			}
			return result

		case Medium:
			result, ok := applyRules(n)
			if !ok {
				continue
			}
			return result

		case Hard:
			result, ok := applyRules(n)
			if !ok {
				continue
			}
			// At least one repeating digit and sum of digits must be prime
			s := strconv.Itoa(result)
			if !hasRepeatingDigit(s) || !isPrime(sumDigits(s)) {
				continue
			}
			return result
		}
	}
}

func GenerateFeedback(secret, guess int) string {
	s := strconv.Itoa(secret)
	g := strconv.Itoa(guess)

	correctCount := 0
	misplacedCount := 0
	firstHalfCorrect := 0
	misplacedFromFirst := 0

	for i := 0; i < 4; i++ {
		if g[i] == s[i] {
			correctCount++
			if i < 2 {
				firstHalfCorrect++
			}
		} else {
			for j := 0; j < 4; j++ {
				if i != j && g[i] == s[j] {
					misplacedCount++
					if i < 2 {
						misplacedFromFirst++
					}
					break
				}
			}
		}
	}

	if correctCount == 4 {
		return "Congratulations! You guessed the correct number!"
	}

	feedback := fmt.Sprintf("Correct digits in right position: %d. Correct digits in wrong position: %d.", correctCount, misplacedCount)

	if correctCount > 0 {
		if firstHalfCorrect > 0 {
			feedback += " Hint: at least one correct digit is in the first half of the number."
		} else {
			feedback += " Hint: all correct digits are in the second half of the number."
		}
	}

	if misplacedCount > 0 {
		if misplacedFromFirst > 0 {
			feedback += " Hint: at least one misplaced digit comes from the first half of your guess."
		} else {
			feedback += " Hint: all misplaced digits come from the second half of your guess."
		}
	}

	return feedback
}

func GenerateTimestampPrefix() string {
	currentTime := time.Now()
	timestamp := currentTime.Unix()
	prefix := "TIME: " + fmt.Sprintf("%d", timestamp)
	return prefix
}
