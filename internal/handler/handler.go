package handler

import (
	"github.com/Prosus-Cyber-Xchange/anonymizer/internal/monitoring"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/config"
	contextpkg "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/context"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"github.com/uber-go/tally/v4"
)

// Handler handles HTTP requests for the anonymizer API
type Handler struct {
	logger         *slog.Logger
	privacyService *privacy.Service
	bufferPool     *BufferPool
	metrics        PrivacyMetrics
	maxBatchSize   int
}

// NewHandler creates a new API handler
func NewHandler(logger *slog.Logger, privacyService *privacy.Service, maxBatchSize int) *Handler {
	// Use NoopScope for default metrics (no-op implementation)
	scope := tally.NoopScope
	metrics := PrivacyMetrics{scope: scope}
	return NewHandlerWithMetrics(logger, privacyService, maxBatchSize, metrics)
}

// NewHandlerWithMetrics creates a new API handler with custom metrics
func NewHandlerWithMetrics(logger *slog.Logger, privacyService *privacy.Service, maxBatchSize int, metrics PrivacyMetrics) *Handler {
	return &Handler{
		logger:         logger,
		privacyService: privacyService,
		bufferPool:     NewBufferPool(),
		metrics:        metrics,
		maxBatchSize:   maxBatchSize,
	}
}

// NOTE: using json.RawMessage can actually be worse than marshalling
// See this issue for more details: https://github.com/golang/go/issues/33422

type ByteString []byte

func (b ByteString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(b))
}

func (b *ByteString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*b = ByteString(s)
	return nil
}

// anonymizeRequest represents the request body for the general anonymize endpoint
type anonymizeRequest struct {
	Text     ByteString              `json:"text"`
	Settings privacy.PrivacySettings `json:"settings"`
}

// anonymizeResponse represents the response body for the general anonymize endpoint
type anonymizeResponse struct {
	AnonymizedText     ByteString `json:"anonymized_text"`
	DetectedEntities   []string   `json:"detected_entities"`
	AnonymizedEntities []string   `json:"anonymized_entities"`
}

// Anonymize dispatches to appropriate handler based on Content-Type
func (h *Handler) Anonymize(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	// Normalize: strip parameters (e.g., "text/plain; charset=utf-8" → "text/plain")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	contentType = strings.ToLower(contentType)

	switch contentType {
	case "text/plain":
		h.anonymizeTextPlain(w, r)
	case "application/json", "":
		h.anonymizeJSON(w, r)
	default:
		respondError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
			fmt.Sprintf("unsupported content type: %s", contentType))
	}
}

// anonymizeJSON handles POST /api/v1/anonymize requests with JSON content and inline privacy settings
func (h *Handler) anonymizeJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	span, ctx := monitoring.StartSpan(ctx, "anonymize_json_request")
	defer span.Finish()

	var req anonymizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "Failed to decode request body", slog.String("error", err.Error()))
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	defer r.Body.Close()

	if err := config.ValidatePrivacyConfig(req.Settings); err != nil {
		h.logger.WarnContext(ctx, "Invalid privacy settings", slog.String("error", err.Error()))
		monitoring.SetError(span, err)
		respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
		return
	}

	ruleSet, err := privacy.NewRuleBuilder(req.Settings).Build()
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to build rules from settings", slog.String("error", err.Error()))
		monitoring.SetError(span, err)
		respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
		return
	}

	responseBuffer := h.bufferPool.GetResponseBuffer()
	defer h.bufferPool.PutResponseBuffer(responseBuffer)

	start := time.Now()
	output, err := h.privacyService.AnonymizeWithRules(ctx, ruleSet, req.Text, responseBuffer)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to anonymize content", slog.String("error", err.Error()))
		monitoring.SetError(span, err)
		respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
		return
	}
	h.metrics.ObserveAnonymizationDuration(time.Since(start))
	for _, entity := range output.Details.AnonymizedEntities {
		h.metrics.CountAnonymizedEntity(string(entity))
	}

	detectedEntities := entitySliceToStrings(output.Details.DetectedEntities)
	anonymizedEntities := entitySliceToStrings(output.Details.AnonymizedEntities)

	resp := anonymizeResponse{
		AnonymizedText:     responseBuffer.Bytes(),
		DetectedEntities:   detectedEntities,
		AnonymizedEntities: anonymizedEntities,
	}

	respondJSON(w, http.StatusOK, resp)
}

// anonymizeTextPlain handles POST /api/v1/anonymize requests with text/plain content
func (h *Handler) anonymizeTextPlain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	span, ctx := monitoring.StartSpan(ctx, "anonymize_textplain_request")
	defer span.Finish()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Build settings from headers
	headerMap := map[string]string{
		HeaderEntities:    r.Header.Get(HeaderEntities),
		HeaderStrategy:    r.Header.Get(HeaderStrategy),
		HeaderPlaceholder: r.Header.Get(HeaderPlaceholder),
		HeaderMaskChar:    r.Header.Get(HeaderMaskChar),
		HeaderMaskLength:  r.Header.Get(HeaderMaskLength),
	}

	var rules []analyzer.Rule

	// Check if headers provide inline settings (highest priority)
	if headerMap[HeaderEntities] != "" {
		settings, err := parseAnonymizeHeaders(headerMap)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_HEADERS", err.Error())
			return
		}
		if err := config.ValidatePrivacyConfig(settings); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
			return
		}
		rulesSlice, err := privacy.NewRuleBuilder(settings).Build()
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
			return
		}
		rules = rulesSlice
	} else {
		// todo: context-injected rules should be supported in all endpoints.  If JSON body `settings is present, it takes priority over context rules. If headers are present, it takes priority over both headers and context rules. This should be clearly documented in the API docs.`
		// Fallback: check context for pre-injected rules (from plugin middleware)
		ctxRules, ok := contextpkg.RulesFromContext(ctx)
		if !ok || len(ctxRules) == 0 {
			respondError(w, http.StatusBadRequest, "NO_RULES",
				"X-Anonymize-Entities header is required when no rules are pre-configured")
			return
		}
		rules = ctxRules
	}

	responseBuffer := h.bufferPool.GetResponseBuffer()
	defer h.bufferPool.PutResponseBuffer(responseBuffer)

	start := time.Now()
	output, err := h.privacyService.AnonymizeWithRules(ctx, rules, body, responseBuffer)
	if err != nil {
		monitoring.SetError(span, err)
		respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
		return
	}
	h.metrics.ObserveAnonymizationDuration(time.Since(start))
	for _, entity := range output.Details.AnonymizedEntities {
		h.metrics.CountAnonymizedEntity(string(entity))
	}

	// Build response headers
	detectedEntities := entitySliceToStrings(output.Details.DetectedEntities)
	anonymizedEntities := entitySliceToStrings(output.Details.AnonymizedEntities)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set(HeaderDetectedEntities, strings.Join(detectedEntities, ","))
	w.Header().Set(HeaderAnonymizedEntities, strings.Join(anonymizedEntities, ","))
	w.WriteHeader(http.StatusOK)
	w.Write(responseBuffer.Bytes())
}

// entitySliceToStrings converts a slice of pattern.Entity to []string and sorts the result
func entitySliceToStrings(entities []pattern.Entity) []string {
	result := make([]string, 0, len(entities))
	for _, e := range entities {
		result = append(result, string(e))
	}
	sort.Strings(result)
	return result
}

// AnonymizeBatch handles POST /api/v1/anonymize/batch requests
func (h *Handler) AnonymizeBatch(w http.ResponseWriter, r *http.Request) {
	// Check content type - batch endpoint only accepts application/json
	contentType := r.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	if contentType != "" && strings.ToLower(contentType) != "application/json" {
		respondError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
			"batch endpoint only accepts application/json")
		return
	}

	ctx := r.Context()

	span, ctx := monitoring.StartSpan(ctx, "anonymize_batch_request")
	defer span.Finish()

	var reqs []anonymizeRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		h.logger.WarnContext(ctx, "Failed to decode batch request body", slog.String("error", err.Error()))
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	defer r.Body.Close()

	if len(reqs) > h.maxBatchSize {
		h.logger.WarnContext(ctx, "Batch size exceeds limit",
			slog.Int("batch_size", len(reqs)),
			slog.Int("max_batch_size", h.maxBatchSize),
		)
		msg := fmt.Sprintf("batch size %d exceeds maximum allowed size of %d", len(reqs), h.maxBatchSize)
		respondError(w, http.StatusBadRequest, "BATCH_SIZE_EXCEEDED", msg)
		return
	}

	responses := make([]anonymizeResponse, 0, len(reqs))
	for i, req := range reqs {
		if err := config.ValidatePrivacyConfig(req.Settings); err != nil {
			h.logger.WarnContext(ctx, "Invalid privacy settings in batch item",
				slog.Int("index", i),
				slog.String("error", err.Error()))
			monitoring.SetError(span, err)
			respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
			return
		}

		ruleSet, err := privacy.NewRuleBuilder(req.Settings).Build()
		if err != nil {
			h.logger.ErrorContext(ctx, "Failed to build rules from settings in batch item",
				slog.Int("index", i),
				slog.String("error", err.Error()))
			monitoring.SetError(span, err)
			respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
			return
		}

		responseBuffer := h.bufferPool.GetResponseBuffer()
		start := time.Now()
		output, err := h.privacyService.AnonymizeWithRules(ctx, ruleSet, req.Text, responseBuffer)
		if err != nil {
			h.bufferPool.PutResponseBuffer(responseBuffer)
			h.logger.ErrorContext(ctx, "Failed to anonymize batch item",
				slog.Int("index", i),
				slog.String("error", err.Error()))
			monitoring.SetError(span, err)
			respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
			return
		}

		h.metrics.ObserveAnonymizationDuration(time.Since(start))
		for _, entity := range output.Details.AnonymizedEntities {
			h.metrics.CountAnonymizedEntity(string(entity))
		}

		// Copy bytes before returning buffer to pool
		anonymizedText := make([]byte, responseBuffer.Len())
		copy(anonymizedText, responseBuffer.Bytes())
		h.bufferPool.PutResponseBuffer(responseBuffer)

		detectedEntities := make([]string, 0, len(output.Details.DetectedEntities))
		for _, ent := range output.Details.DetectedEntities {
			detectedEntities = append(detectedEntities, string(ent))
		}
		sort.Strings(detectedEntities)

		anonymizedEntities := make([]string, 0, len(output.Details.AnonymizedEntities))
		for _, ent := range output.Details.AnonymizedEntities {
			anonymizedEntities = append(anonymizedEntities, string(ent))
		}
		sort.Strings(anonymizedEntities)

		responses = append(responses, anonymizeResponse{
			AnonymizedText:     anonymizedText,
			DetectedEntities:   detectedEntities,
			AnonymizedEntities: anonymizedEntities,
		})
	}

	respondJSON(w, http.StatusOK, responses)
}
