package ezdb

import "errors"

var (
	ErrInvalidDBDriver   = errors.New("invalid db driver")
	ErrInvalidDBLogLevel = errors.New("invalid db log level")
)
