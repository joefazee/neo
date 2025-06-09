package formatter

import (
	"strings"

	"github.com/nyaruka/phonenumbers"
)

// FormatPhone formats a phone number to E164 format
func FormatPhone(phone, countryCode string) (string, error) {
	countryCode = strings.ToUpper(countryCode)
	num, err := phonenumbers.Parse(phone, countryCode)
	if err != nil {
		return "", err
	}
	formattedNum := phonenumbers.Format(num, phonenumbers.E164)
	return formattedNum, nil
}
