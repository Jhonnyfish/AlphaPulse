package services

import (
	"fmt"
	"regexp"
	"strings"
)

// A-share stock code patterns:
//   600xxx, 601xxx, 603xxx, 605xxx — Shanghai main board
//   688xxx — Shanghai STAR market (科创板)
//   000xxx, 001xxx — Shenzhen main board
//   002xxx — Shenzhen SME board (中小板)
//   300xxx, 301xxx — Shenzhen ChiNext (创业板)
//   83xxxx, 87xxxx — Beijing Stock Exchange (北交所)
//   900xxx — Shanghai B-shares
//   200xxx — Shenzhen B-shares

var validCodePattern = regexp.MustCompile(`^\d{6}$`)

// ShanghaiPrefixes are codes that trade on Shanghai exchange.
var ShanghaiPrefixes = []string{"600", "601", "603", "605", "688", "900"}

// ValidateStockCode checks if a code is a plausible A-share stock code.
// Returns error describing why the code is invalid, or nil if valid.
func ValidateStockCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("stock code is empty")
	}
	if !validCodePattern.MatchString(code) {
		return fmt.Errorf("stock code %q must be exactly 6 digits", code)
	}

	// Check known prefixes
	prefix := code[:3]
	knownPrefixes := []string{
		"600", "601", "603", "605", "688", // Shanghai
		"000", "001", "002", "300", "301", // Shenzhen
		"830", "831", "832", "833", "834", "835", "836", "837", "838", "839", // Beijing
		"870", "871", "872", "873", "874", "875", "876", "877", "878", "879", // Beijing
		"900", // Shanghai B
		"200", // Shenzhen B
		"430", "431", "432", "433", "434", "435", "436", "437", "438", "439", // Beijing old
	}

	for _, p := range knownPrefixes {
		if prefix == p {
			return nil
		}
	}

	// Not a known prefix — still allow it (might be a new listing)
	// but log a warning-worthy pattern
	return nil
}

// IsShanghai returns true if the code is a Shanghai-listed stock.
func IsShanghai(code string) bool {
	for _, p := range ShanghaiPrefixes {
		if strings.HasPrefix(code, p) {
			return true
		}
	}
	return false
}

// EastMoneySecID converts a stock code to EastMoney's secid format.
// Shanghai: "1.CODE", Shenzhen: "0.CODE"
func EastMoneySecID(code string) string {
	if IsShanghai(code) {
		return "1." + code
	}
	return "0." + code
}

// TencentSymbol converts a stock code to Tencent's symbol format.
// Shanghai: "shCODE", Shenzhen: "szCODE"
func TencentSymbol(code string) string {
	if IsShanghai(code) {
		return "sh" + code
	}
	return "sz" + code
}
