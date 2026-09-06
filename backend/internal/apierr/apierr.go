// Package apierr is how handlers report a failure to the client.
//
// An error response carries a machine-readable `error_code` and an English
// `error` message. The code is what the UI renders — the frontend looks up
// t("errors.<code>") and shows it in the user's language — so the server never
// has to know which language the caller reads. The message is developer-facing:
// it is what appears in logs, in curl output and in the fallback path, and it
// stays in English on purpose.
//
// This replaces handlers returning localized prose. Those strings were a mix of
// Italian and English, so no user could get a consistent language, and adding a
// third language would have meant a second translation pipeline on the server.
//
// Internal failures all report CodeServerError: their specific message is a
// diagnostic ("Failed to delete expense splits"), not something a user can act
// on, so the UI shows one generic sentence while the detail stays in `error`
// for whoever is debugging.
package apierr

import (
	"github.com/gin-gonic/gin"
)

// CodeServerError is the code every unexpected internal failure reports.
const CodeServerError = "server_error"

// CodeInvalidRequest is the code for a request the server could not parse or
// bind — the detail comes from the validator and is developer-facing.
const CodeInvalidRequest = "invalid_request"

// Response is the body of every error reply.
type Response struct {
	// Error is the English, developer-facing message.
	Error string `json:"error"`
	// Code is the stable identifier the client translates.
	Code string `json:"error_code"`
	// Params fills placeholders in the translated message, when it has any.
	Params map[string]string `json:"error_params,omitempty"`
}

// Fail writes an error response with the given status.
//
// code must exist in frontend/src/i18n/locales/*/errors.json; a test enforces
// that, because a code with no translation would render as a raw key.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, Response{Error: message, Code: code})
}

// FailWith is Fail for a message whose translation takes placeholders, e.g.
// "Currency {currency} is not supported".
func FailWith(c *gin.Context, status int, code, message string, params map[string]string) {
	c.JSON(status, Response{Error: message, Code: code, Params: params})
}
