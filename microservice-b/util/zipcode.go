package util

import (
	"errors"
	"regexp"
	"strings"
)

const regexZipCodeWithHyphen = `^\d{5}-\d{3}$` // nnnnn-nnn format
const regexZipCodeWithoutHyphen = `^\d{8}$`    // nnnnnnnn format
var zipCodeValidatorWithHyphen = regexp.MustCompile(regexZipCodeWithHyphen)
var zipCodeValidatorWithoutHyphen = regexp.MustCompile(regexZipCodeWithoutHyphen)

func FormatZipCode(zipCode string) (string, error) {
	if zipCodeValidatorWithHyphen.MatchString(zipCode) {
		// this pattern works for both ViaCEP and ApiCEP
		return zipCode, nil
	}
	if zipCodeValidatorWithoutHyphen.MatchString(zipCode) {
		// including hyphen, since this pattern works for both ViaCEP and ApiCEP
		return includeHyphen(zipCode), nil
	}
	return "", errors.New("invalid zip code")
}

func includeHyphen(zipcode string) string {
	return strings.Join([]string{zipcode[:5], zipcode[5:]}, "-")
}
