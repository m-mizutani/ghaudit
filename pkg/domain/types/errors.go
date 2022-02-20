package types

import "github.com/m-mizutani/goerr"

var (
	ErrNoEvalResult = goerr.New("no eval result")

	ErrUnexpectedGitHubResp = goerr.New("unexpected github response")

	ErrInvalidConfig = goerr.New("invalid config")

	ErrViolationDetected = goerr.New("violation detected")
)
