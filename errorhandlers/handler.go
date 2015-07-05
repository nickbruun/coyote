package errorhandlers

// Error handler.
type Handler interface {
	// Handle error.
	Handle(err *Error) error
}
