package node

import (
	"log"
	"sort"
	"strings"
)

func logNetworkEvent(event string, fields map[string]string) {
	var b strings.Builder
	b.WriteString("event=")
	b.WriteString(sanitizeLogValue(event))
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := fields[key]
		if strings.TrimSpace(key) == "" {
			continue
		}
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(sanitizeLogValue(value))
	}
	log.Print(b.String())
}

func sanitizeLogValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "\"\""
	}
	value = strings.ReplaceAll(value, "\"", "'")
	value = strings.ReplaceAll(value, "\n", " ")
	return "\"" + value + "\""
}
