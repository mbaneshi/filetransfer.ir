package auth

import (
	"errors"
	"strings"
)

var ErrInvalidPhone = errors.New("invalid iranian mobile number")

// NormalizePhone turns the many ways people type an Iranian mobile number
// (Persian digits, 0912…, +98912…, 0098912…) into E.164, e.g. +989121234567.
func NormalizePhone(input string) (string, error) {
	var digits strings.Builder
	for _, r := range input {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r >= '۰' && r <= '۹': // Persian digits
			digits.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩': // Arabic-Indic digits
			digits.WriteRune('0' + (r - '٠'))
		case r == ' ' || r == '-' || r == '+' || r == '(' || r == ')':
		default:
			return "", ErrInvalidPhone
		}
	}

	n := digits.String()
	switch {
	case strings.HasPrefix(n, "0098"):
		n = n[4:]
	case strings.HasPrefix(n, "98"):
		n = n[2:]
	case strings.HasPrefix(n, "0"):
		n = n[1:]
	}
	if len(n) != 10 || n[0] != '9' {
		return "", ErrInvalidPhone
	}
	return "+98" + n, nil
}

// LocalFormat returns the 09xxxxxxxxx form that Iranian SMS gateways expect.
func LocalFormat(e164 string) string {
	return "0" + strings.TrimPrefix(e164, "+98")
}
