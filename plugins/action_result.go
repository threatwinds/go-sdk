package plugins

// Values for Event.ActionResult, the final outcome of the action an event
// describes. Filters write exactly one of ActionResultSuccess,
// ActionResultFailed or ActionResultDenied, in lowercase, or leave the field
// empty when the record states no final outcome. Consumers compare these words
// exactly, so a vendor's own word (Success, accepted, fail, done) must be
// translated, not copied.
//
// The event processor's threat-intelligence analysis (the feeds plugin) does
// not look up an event's indicators when its ActionResult is failed, denied or
// blocked: a failed or refused attempt is not a threat. Any other value,
// including an empty one, is looked up.
const (
	// ActionResultSuccess means the action completed: the sign-in succeeded,
	// the connection was allowed and answered, the request was served.
	ActionResultSuccess = "success"
	// ActionResultFailed means the action was attempted and did not complete:
	// wrong password, locked account, error, timeout, failed negotiation.
	ActionResultFailed = "failed"
	// ActionResultDenied means a control refused the action: firewall deny,
	// drop or reject, access or policy denial, security block, quarantine.
	ActionResultDenied = "denied"
	// ActionResultBlocked is read by the event processor as ActionResultDenied
	// so that older filters keep working. Filters must write
	// ActionResultDenied instead, so rules test a single word.
	ActionResultBlocked = "blocked"
)

// IsActionResult reports whether value is one a filter may write to
// Event.ActionResult: success, failed, denied, or empty.
func IsActionResult(value string) bool {
	switch value {
	case "", ActionResultSuccess, ActionResultFailed, ActionResultDenied:
		return true
	default:
		return false
	}
}

// IsUnsuccessfulActionResult reports whether value states that the action did
// not complete (failed, denied, or the older blocked). The event processor's
// threat-intelligence analysis skips indicator lookups for these values.
func IsUnsuccessfulActionResult(value string) bool {
	switch value {
	case ActionResultFailed, ActionResultDenied, ActionResultBlocked:
		return true
	default:
		return false
	}
}
