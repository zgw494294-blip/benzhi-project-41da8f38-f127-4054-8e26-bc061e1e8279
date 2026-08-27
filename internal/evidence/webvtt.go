package evidence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func BuildWebVTT(revision domain.CaptionRevision) ([]byte, string) {
	segments := append([]domain.Segment(nil), revision.Segments...)
	sort.SliceStable(segments, func(i, j int) bool {
		if segments[i].StartMillis == segments[j].StartMillis {
			return segments[i].ID < segments[j].ID
		}
		return segments[i].StartMillis < segments[j].StartMillis
	})
	var out bytes.Buffer
	out.WriteString("WEBVTT\n\n")
	for _, segment := range segments {
		fmt.Fprintf(&out, "%s\n%s --> %s\n", segment.ID, vttTime(segment.StartMillis), vttTime(segment.EndMillis))
		text := strings.ReplaceAll(segment.Text, "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
		if strings.TrimSpace(segment.Speaker) != "" {
			fmt.Fprintf(&out, "<v %s>%s\n\n", segment.Speaker, text)
		} else {
			fmt.Fprintf(&out, "%s\n\n", text)
		}
	}
	data := out.Bytes()
	return data, SHA256(data)
}

func vttTime(ms int64) string {
	hours := ms / 3_600_000
	ms %= 3_600_000
	minutes := ms / 60_000
	ms %= 60_000
	seconds := ms / 1000
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}
