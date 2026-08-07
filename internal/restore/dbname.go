package restore

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reCreateDatabase = regexp.MustCompile("(?i)CREATE\\s+DATABASE(?:\\s+IF\\s+NOT\\s+EXISTS)?\\s+(?:`([^`]+)`|([A-Za-z0-9_$]+))")
	reUseDatabase    = regexp.MustCompile("(?i)USE\\s+(?:`([^`]+)`|([A-Za-z0-9_$]+))\\s*;")
)

// DefaultPeekBytes is how much of a dump to read when extracting the database name.
const DefaultPeekBytes int64 = 128 * 1024

// ExtractDatabaseName finds the target database from a SQL dump prefix.
// Prefers CREATE DATABASE over USE. Returns an error if neither is found.
func ExtractDatabaseName(sqlPrefix []byte) (string, error) {
	text := string(sqlPrefix)
	if name := firstSubmatch(reCreateDatabase.FindStringSubmatch(text)); name != "" {
		return name, nil
	}
	if name := firstSubmatch(reUseDatabase.FindStringSubmatch(text)); name != "" {
		return name, nil
	}
	return "", fmt.Errorf("dump prefix has no CREATE DATABASE or USE statement")
}

func firstSubmatch(m []string) string {
	if len(m) < 3 {
		return ""
	}
	if m[1] != "" {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(m[2])
}
