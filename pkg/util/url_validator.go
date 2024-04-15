package util

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func ParseURL(u string) error {
	_, err := url.ParseRequestURI(u)
	return err
}

func isValidScheme(inputScheme string, validSchemes []string) bool {
	for _, scheme := range validSchemes {
		if strings.EqualFold(inputScheme, scheme) {
			return true
		}
	}
	return false
}

func ValidateURL(u string) error {
	validSchemes := []string{"http", "https", "ftp"}

	// Parse the URL
	parsedURL, err := url.Parse(u)
	if err != nil {
		return err
	}

	// Check if the scheme is empty or not
	if parsedURL.Scheme == "" {
		return errors.New("URL Scheme is empty")
	}
	if !isValidScheme(parsedURL.Scheme, validSchemes) {
		return fmt.Errorf("invalid URL Scheme %v", parsedURL.Scheme)
	}

	// Check if the host is empty or not
	if parsedURL.Host == "" {
		return errors.New("URL Host is empty")
	}

	return nil
}
