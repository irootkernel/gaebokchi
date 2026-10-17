package domain

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

const (
	maxNumericTokenBytes = 128
	maxCoefficientDigits = 64
	maxExplicitExponent  = 128
	maxFractionalScale   = 128
)

// ValidatePercent reports whether v is an acceptable provider-reported
// percentage: finite and within [0,100]. Values outside the range are a
// protocol/semantic mismatch and must not be clamped or substituted.
func ValidatePercent(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v >= 0 && v <= 100
}

// NormalizePercentSignedZero maps -0 to 0 so formatting never emits "-0%".
func NormalizePercentSignedZero(v float64) float64 {
	if v == 0 {
		return 0
	}
	return v
}

// RemainingFractionToUsedPercent converts a JSON number token representing a
// remaining fraction r in [0,1] into used_percent = 100 * (1 - r) using bounded
// exact decimal arithmetic before converting to float64. This prevents binary
// floating-point subtraction noise (e.g. 100*(1-0.9) = 9.999999999999998).
func RemainingFractionToUsedPercent(token string) (float64, bool) {
	if len(token) == 0 || len(token) > maxNumericTokenBytes {
		return 0, false
	}
	s := token
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
		if s == "" {
			return 0, false
		}
	}
	mantissa := s
	expStr := ""
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mantissa = s[:i]
		expStr = s[i+1:]
		if expStr == "" {
			return 0, false
		}
	}
	intPart := mantissa
	fracPart := ""
	if dot := strings.IndexByte(mantissa, '.'); dot >= 0 {
		if strings.IndexByte(mantissa[dot+1:], '.') >= 0 {
			return 0, false
		}
		intPart = mantissa[:dot]
		fracPart = mantissa[dot+1:]
		if intPart == "" || fracPart == "" {
			return 0, false
		}
	} else if intPart == "" {
		return 0, false
	}
	if !isAllASCIIDigits(intPart) || !isAllASCIIDigits(fracPart) {
		return 0, false
	}
	rawDigits := strings.TrimLeft(intPart+fracPart, "0")
	if len(rawDigits) > maxCoefficientDigits {
		return 0, false
	}
	exp := 0
	if expStr != "" {
		expSign := 1
		if expStr[0] == '-' || expStr[0] == '+' {
			if expStr[0] == '-' {
				expSign = -1
			}
			expStr = expStr[1:]
		}
		if expStr == "" || !isAllASCIIDigits(expStr) {
			return 0, false
		}
		trimmedExp := strings.TrimLeft(expStr, "0")
		if len(trimmedExp) > 4 {
			return 0, false
		}
		if trimmedExp != "" {
			v, err := strconv.Atoi(trimmedExp)
			if err != nil {
				return 0, false
			}
			exp = expSign * v
		}
		if exp < -maxExplicitExponent || exp > maxExplicitExponent {
			return 0, false
		}
	}
	if rawDigits == "" {
		// Exact zero (including -0, 0.0, 0e-5) -> 100% used.
		return 100, true
	}
	if neg {
		return 0, false
	}
	scale := len(fracPart) - exp
	// Strip trailing zeros in rawDigits to minimize scale before checking bounds.
	for len(rawDigits) > 1 && rawDigits[len(rawDigits)-1] == '0' {
		rawDigits = rawDigits[:len(rawDigits)-1]
		scale--
	}
	if scale > maxFractionalScale {
		return 0, false
	}
	coeff := new(big.Int)
	if _, ok := coeff.SetString(rawDigits, 10); !ok {
		return 0, false
	}
	if scale < 0 {
		// Value is coeff * 10^(-scale), which for coeff >= 1 and -scale >= 1 is >= 10 > 1.
		return 0, false
	}
	pow10 := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	if coeff.Cmp(pow10) > 0 {
		return 0, false
	}
	if coeff.Cmp(pow10) == 0 {
		// Remaining fraction == 1 -> 0% used.
		return 0, true
	}
	// used = 100 * (pow10 - coeff) / pow10 = (pow10 - coeff) / 10^(scale - 2).
	rem := new(big.Int).Sub(pow10, coeff)
	usedScale := scale - 2
	var decStr string
	if usedScale <= 0 {
		mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-usedScale)), nil)
		decStr = new(big.Int).Mul(rem, mul).String()
	} else {
		digits := rem.String()
		if len(digits) <= usedScale {
			digits = strings.Repeat("0", usedScale-len(digits)+1) + digits
		}
		split := len(digits) - usedScale
		intStr := digits[:split]
		fracStr := strings.TrimRight(digits[split:], "0")
		if fracStr == "" {
			decStr = intStr
		} else {
			decStr = intStr + "." + fracStr
		}
	}
	v, err := strconv.ParseFloat(decStr, 64)
	if err != nil || !ValidatePercent(v) {
		return 0, false
	}
	return NormalizePercentSignedZero(v), true
}

func isAllASCIIDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
