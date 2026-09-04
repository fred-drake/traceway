package main

import (
	"fmt"
	"slices"
	"strings"
)

// validateEnumFlag checks that value is one of allowed. On mismatch it returns
// an error formatted for the usage_error envelope's Message field. The flag
// argument is the user-facing flag name (e.g. "--aggregation") and is included
// in the message so the user knows which input was rejected.
func validateEnumFlag(flag, value string, allowed []string) error {
	if slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("%s must be one of: %s", flag, strings.Join(allowed, ", "))
}

// enumFlagHint returns a usage_error hint string suggesting the canonical form
// for an enum flag — e.g. "traceway metrics query --aggregation <avg|sum|...>".
func enumFlagHint(cmdPath, flag string, allowed []string) string {
	return fmt.Sprintf("%s %s <%s>", cmdPath, flag, strings.Join(allowed, "|"))
}
