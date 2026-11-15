package app

import (
	"errors"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
)

var (
	// ErrUnknownAct tells that the source has no act with the identifier.
	ErrUnknownAct = errors.New("act: unknown")
	// ErrNoTranslation tells that the source has no English text of the act or the version.
	ErrNoTranslation = errors.New("act: no English translation")
	// ErrUnknownVersion tells that the act has no such version.
	ErrUnknownVersion = errors.New("version: unknown")
	// ErrTooManyArticles tells that a query asks more than 10 articles.
	ErrTooManyArticles = errors.New("articles: more than 10")
	// ErrBothVersions tells that a query gives a version number and a date.
	ErrBothVersions = errors.New("version: a number and a date")
	// ErrUnknownArticle tells that the act has no such article.
	ErrUnknownArticle = errors.New("article: unknown")
	// ErrUnknownPiece tells that the text has no such piece.
	ErrUnknownPiece = errors.New("piece: unknown")
)

// ArticleError tells that the act has no article with the number. First and Last are the
// first and the last article of the act, and both are zero when the act has no article.
type ArticleError struct {
	Number, First, Last domain.ArticleNumber
}

// Error returns the text of ErrUnknownArticle.
func (e *ArticleError) Error() string {
	return ErrUnknownArticle.Error()
}

// Unwrap returns ErrUnknownArticle.
func (e *ArticleError) Unwrap() error {
	return ErrUnknownArticle
}

// VersionError tells that the act has no version that matches the query. First is version
// 0 and Newest is the newest version.
type VersionError struct {
	First, Newest domain.Version
}

// Error returns the text of ErrUnknownVersion.
func (e *VersionError) Error() string {
	return ErrUnknownVersion.Error()
}

// Unwrap returns ErrUnknownVersion.
func (e *VersionError) Unwrap() error {
	return ErrUnknownVersion
}

// PieceError tells that the text has no piece with the number. Pieces is the count.
type PieceError struct {
	Piece, Pieces int
}

// Error returns the text of ErrUnknownPiece.
func (e *PieceError) Error() string {
	return ErrUnknownPiece.Error()
}

// Unwrap returns ErrUnknownPiece.
func (e *PieceError) Unwrap() error {
	return ErrUnknownPiece
}
