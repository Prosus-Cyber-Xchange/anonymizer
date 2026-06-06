package e2e_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/e2e/driver"
	"github.com/Prosus-Cyber-Xchange/anonymizer/e2e/specifications"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
)

func TestAnonymizer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Wire the application (no external containers needed — no DB/Redis/external APIs)
	svc, err := server.NewFromConfig(context.Background())
	if err != nil {
		t.Fatalf("failed to create anonymizer service: %v", err)
	}
	srv := httptest.NewServer(svc.Handler())
	t.Cleanup(srv.Close)

	d := driver.NewHTTPDriver(srv.URL, srv.Client())

	// Run specifications
	t.Run("health", specifications.HealthCheck(d))

	t.Run("anonymize JSON success", specifications.AnonymizeJSON_Success(d))
	t.Run("anonymize JSON multiple entities", specifications.AnonymizeJSON_MultipleEntities(d))
	t.Run("anonymize JSON no entities detected", specifications.AnonymizeJSON_NoEntitiesDetected(d))
	t.Run("anonymize JSON invalid settings", specifications.AnonymizeJSON_InvalidSettings(d))
	t.Run("anonymize JSON no content-type defaults to JSON", specifications.AnonymizeJSON_NoContentTypeDefaultsToJSON(d))

	t.Run("anonymize text/plain success", specifications.AnonymizeTextPlain_Success(d))
	t.Run("anonymize text/plain entity header with spaces", specifications.AnonymizeTextPlain_EntityHeaderWithSpaces(d))
	t.Run("anonymize text/plain missing entities header", specifications.AnonymizeTextPlain_MissingEntitiesHeader(d))

	t.Run("batch multiple items", specifications.AnonymizeBatch_MultipleItems(d))
	t.Run("batch exceeds max size", specifications.AnonymizeBatch_ExceedsMaxSize(d))
	t.Run("batch unsupported media type", specifications.AnonymizeBatch_UnsupportedMediaType(d))
}
