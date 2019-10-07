package github

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type refSelector struct {
	WhitelistedRefs []string `json:"only,omitempty"`
	BlacklistedRefs []string `json:"ignore,omitempty"`
}

func (r *refSelector) matches(ref string) (bool, error) {
	var matchesWhitelist bool
	if len(r.WhitelistedRefs) == 0 {
		matchesWhitelist = true
	} else {
		for _, whitelistedRef := range r.WhitelistedRefs {
			var err error
			matchesWhitelist, err = refMatch(ref, whitelistedRef)
			if err != nil {
				return false, err
			}
			if matchesWhitelist {
				break
			}
		}
	}
	var matchesBlacklist bool
	for _, blacklistedRef := range r.BlacklistedRefs {
		var err error
		matchesBlacklist, err = refMatch(ref, blacklistedRef)
		if err != nil {
			return false, err
		}
		if matchesBlacklist {
			break
		}
	}
	return matchesWhitelist && !matchesBlacklist, nil
}

func refMatch(ref, valueOrPattern string) (bool, error) {
	if strings.HasPrefix(valueOrPattern, "/") &&
		strings.HasSuffix(valueOrPattern, "/") {
		pattern := valueOrPattern[1 : len(valueOrPattern)-1]
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return false, errors.Wrapf(
				err,
				"error compiling regular expression %s",
				valueOrPattern,
			)
		}
		return regex.MatchString(ref), nil
	}
	return ref == valueOrPattern, nil
}
