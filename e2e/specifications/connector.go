package specifications

import "context"

// Response is the protocol-agnostic response object.
type Response struct {
	StatusCode int
	Body       string              // JSON string or plain text
	Headers    map[string][]string // HTTP headers or gRPC trailing metadata
}

// AnonymizerClient is the protocol-agnostic interface that drivers must implement.
type AnonymizerClient interface {
	Anonymize(ctx context.Context, body string, headers map[string][]string) (Response, error)
	AnonymizeBatch(ctx context.Context, body string, headers map[string][]string) (Response, error)
	Health(ctx context.Context) (Response, error)
}
