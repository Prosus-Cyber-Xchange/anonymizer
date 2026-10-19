package monitoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// tokenDigestPrefix returns the first 12 hex characters of the SHA-256 digest
// of the token. Spans are tagged with this non-reversible prefix so raw token
// text — unsanitized PII candidate material — is never recorded in APM, while
// token-granular span correlation is preserved.
func tokenDigestPrefix(token string) string {
	digest := sha256.Sum256([]byte(token))

	return hex.EncodeToString(digest[:])[:12]
}

// TraceCacheGetMatches starts a DataDog APM span for a token-scoped cache read
// (GetMatches) and returns a context with the span attached and a finish
// function. The caller must call the finish function when the operation
// completes.
func TraceCacheGetMatches(ctx context.Context, token string) (context.Context, func()) {
	newCtx, span := GlobalTracer().StartSpanFromContext(ctx, "anonymize_cache_get_matches")
	span.SetTag("cache.token", tokenDigestPrefix(token))

	return newCtx, func() {
		span.Finish()
	}
}

// TraceCacheSaveMatches starts a DataDog APM span for a token-scoped cache
// write (SaveMatches) and returns a context with the span attached and a
// finish function. The caller must call the finish function when the
// operation completes.
func TraceCacheSaveMatches(ctx context.Context, token string) (context.Context, func()) {
	newCtx, span := GlobalTracer().StartSpanFromContext(ctx, "anonymize_cache_save_matches")
	span.SetTag("cache.token", tokenDigestPrefix(token))

	return newCtx, func() {
		span.Finish()
	}
}
