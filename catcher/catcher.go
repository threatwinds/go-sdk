package catcher

import "os"

var beauty bool

const (
	debugIcon    = "🔍"  // magnifying glass
	infoIcon     = "ℹ️" // information
	noticeIcon   = "📢"  // loudspeaker
	warningIcon  = "⚠️" // warning
	errorIcon    = "❌"  // cross mark
	criticalIcon = "🔥"  // fire
	alertIcon    = "🚨"  // rotating light
)

func init() {
	b := os.Getenv("CATCHER_BEAUTY")
	if b == "true" {
		beauty = true
	}
}

// GetSeverityIcon returns an icon based on the severity level
func GetSeverityIcon(severity string) string {
	switch severity {
	case "DEBUG":
		return debugIcon
	case "INFO":
		return infoIcon
	case "NOTICE":
		return noticeIcon
	case "WARNING":
		return warningIcon
	case "ERROR":
		return errorIcon
	case "CRITICAL":
		return criticalIcon
	case "ALERT":
		return alertIcon
	default:
		return errorIcon
	}
}
