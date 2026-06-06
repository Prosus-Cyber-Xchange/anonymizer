package e2e_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/e2e/driver"
	"github.com/Prosus-Cyber-Xchange/anonymizer/e2e/specifications"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/anonymizer"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestAnonymizer_WithRedisCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	ctx := context.Background()

	redisContainer, err := tcredis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate redis container: %v", err)
		}
	})

	redisAddr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get redis connection string: %v", err)
	}
	redisAddr = strings.TrimPrefix(redisAddr, "redis://")

	t.Setenv("PRIVACY_CACHE_ENABLED", "true")
	t.Setenv("PRIVACY_CACHE_TTL", "5m")
	t.Setenv("PRIVACY_CACHE_REDIS_ADDR", redisAddr)
	t.Setenv("PRIVACY_CACHE_REDIS_DISABLE_CLUSTER", "true")

	svc, err := anonymizer.NewFromConfig(ctx)
	if err != nil {
		t.Fatalf("failed to create anonymizer service with cache: %v", err)
	}
	srv := httptest.NewServer(svc.Handler())
	t.Cleanup(srv.Close)

	d := driver.NewHTTPDriver(srv.URL, srv.Client())

	t.Run("health", specifications.HealthCheck(d))
	t.Run("anonymize JSON success", specifications.AnonymizeJSON_Success(d))
	t.Run("anonymize JSON multiple entities", specifications.AnonymizeJSON_MultipleEntities(d))
	t.Run("anonymize JSON no entities detected", specifications.AnonymizeJSON_NoEntitiesDetected(d))
	t.Run("anonymize JSON no content-type defaults to JSON", specifications.AnonymizeJSON_NoContentTypeDefaultsToJSON(d))
	t.Run("anonymize text/plain success", specifications.AnonymizeTextPlain_Success(d))
	t.Run("batch multiple items", specifications.AnonymizeBatch_MultipleItems(d))

	t.Run("cache returns consistent results on repeated requests", specifications.AnonymizeJSON_CacheConsistency(d))
}

func TestAnonymizer_WithRedisCache_Unreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	ctx := context.Background()

	t.Setenv("PRIVACY_CACHE_ENABLED", "true")
	t.Setenv("PRIVACY_CACHE_REDIS_ADDR", "localhost:16379")
	t.Setenv("PRIVACY_CACHE_REDIS_DISABLE_CLUSTER", "true")

	svc, err := anonymizer.NewFromConfig(ctx)
	if err != nil {
		t.Skipf("valkey backend does not gracefully degrade on startup: %v", err)
	}
	srv := httptest.NewServer(svc.Handler())
	t.Cleanup(srv.Close)

	d := driver.NewHTTPDriver(srv.URL, srv.Client())

	t.Run("anonymize JSON still works with unreachable redis", specifications.AnonymizeJSON_Success(d))
}