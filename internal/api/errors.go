package api

import (
	"errors"
	"fmt"
)

// Sentinel errors for common API failures. Match them with errors.Is.
var (
	// ErrUnauthorized means the token is invalid or expired beyond refresh.
	ErrUnauthorized = errors.New("não autorizado: execute `42 login` novamente")
	// ErrForbidden means the token lacks the required scope or role.
	ErrForbidden = errors.New("acesso negado: o token não tem permissão para este recurso")
	// ErrNotFound means the requested resource does not exist.
	ErrNotFound = errors.New("recurso não encontrado")
	// ErrRateLimited means the API rate limit was exceeded even after retries.
	ErrRateLimited = errors.New("rate limit da API 42 excedido: aguarde e tente novamente")
	// ErrServer means the API returned a 5xx response after retries.
	ErrServer = errors.New("erro interno da API 42")
)

// Error describes a non-2xx API response.
// It unwraps to one of the sentinel errors above when applicable.
type Error struct {
	StatusCode int
	Body       string
	sentinel   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.sentinel != nil {
		if e.Body != "" {
			return fmt.Sprintf("%s (HTTP %d: %s)", e.sentinel, e.StatusCode, e.Body)
		}
		return fmt.Sprintf("%s (HTTP %d)", e.sentinel, e.StatusCode)
	}
	return fmt.Sprintf("resposta inesperada da API 42: HTTP %d: %s", e.StatusCode, e.Body)
}

// Unwrap exposes the sentinel error for errors.Is matching.
func (e *Error) Unwrap() error {
	return e.sentinel
}

// newHTTPError maps a status code to a typed Error.
func newHTTPError(status int, body string) *Error {
	err := &Error{StatusCode: status, Body: body}
	switch {
	case status == 401:
		err.sentinel = ErrUnauthorized
	case status == 403:
		err.sentinel = ErrForbidden
	case status == 404:
		err.sentinel = ErrNotFound
	case status == 429:
		err.sentinel = ErrRateLimited
	case status >= 500:
		err.sentinel = ErrServer
	}
	return err
}
