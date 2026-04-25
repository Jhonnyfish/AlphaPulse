package models

import (
	"fmt"
	"regexp"
)

var stockCodePattern = regexp.MustCompile(`^\d{6}$`)

// ValidateStockCode checks if a code is a plausible A-share stock code.
func ValidateStockCode(code string) error {
	if code == "" {
		return fmt.Errorf("stock code is empty")
	}
	if !stockCodePattern.MatchString(code) {
		return fmt.Errorf("stock code %q must be exactly 6 digits", code)
	}
	return nil
}
