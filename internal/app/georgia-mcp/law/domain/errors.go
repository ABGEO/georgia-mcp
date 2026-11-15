package domain

import "errors"

var (
	// ErrInvalidActID tells that a text is not an act identifier.
	ErrInvalidActID = errors.New("act identifier: invalid")
	// ErrInvalidArticleNumber tells that a text is not an article number.
	ErrInvalidArticleNumber = errors.New("article number: invalid")
	// ErrUnknownDocumentType tells that a text is not a document type.
	ErrUnknownDocumentType = errors.New("document type: unknown")
	// ErrUnknownActStatus tells that a text is not an act status.
	ErrUnknownActStatus = errors.New("act status: unknown")
)
