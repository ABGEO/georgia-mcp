package mcpserver

// ToolError is the failure of a tool. The model reads Message. Cause stays in the
// server, for the log and for errors.Is.
type ToolError struct {
	Message string
	Cause   error
}

// Error returns the message for the model.
func (e *ToolError) Error() string {
	return e.Message
}

// Unwrap returns the cause.
func (e *ToolError) Unwrap() error {
	return e.Cause
}
