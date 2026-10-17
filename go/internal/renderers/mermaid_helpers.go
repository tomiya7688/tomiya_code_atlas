package renderers

import (
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
)

var nonNodeID = regexp.MustCompile(`[^0-9A-Za-z_]`)

func hashedID(prefix, name string) string {
	digest := sha1.Sum([]byte(name))
	return fmt.Sprintf("%s%x", prefix, digest[:5])
}

func escapeQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, `"`, `\"`)
}

func escapeText(value string) string {
	value = escapeQuote(value)
	return strings.ReplaceAll(value, "\n", " ")
}
