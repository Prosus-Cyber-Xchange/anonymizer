package tlscert

import "errors"

// ErrHostRequired is returned when the host is required but not provided.
var ErrHostRequired = errors.New("host is required")
