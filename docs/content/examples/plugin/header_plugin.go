package main

import (
	"log/slog"
	"net/http"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"
)

// ServiceRulesPlugin demonstrates a plugin that reads a service name header,
// looks up rules for that service, and injects them into the request context.
type ServiceRulesPlugin struct {
	// rulesByService simulates a rule config store (in practice, this would be an API client).
	rulesByService map[string]privacy.PrivacySettings
}

func NewServiceRulesPlugin() *ServiceRulesPlugin {
	return &ServiceRulesPlugin{
		rulesByService: map[string]privacy.PrivacySettings{
			"email-service": {
				Entities: []privacy.EntitySettings{
					{Name: "EMAIL", Redaction: &privacy.RedactionSettings{Replacement: "<EMAIL>"}},
					{Name: "CPF_NUMBER", Redaction: &privacy.RedactionSettings{Replacement: "<CPF>"}},
				},
			},
			"payment-service": {
				Entities: []privacy.EntitySettings{
					{Name: "CREDIT_CARD", Redaction: &privacy.RedactionSettings{Replacement: "<CARD>"}},
				},
			},
		},
	}
}

func (p *ServiceRulesPlugin) Middleware(services server.CoreServices) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceName := r.Header.Get("X-Service-Name")
			if serviceName == "" {
				next.ServeHTTP(w, r)
				return
			}

			settings, ok := p.rulesByService[serviceName]
			if !ok {
				services.Logger.Warn("unknown service", slog.String("service", serviceName))
				next.ServeHTTP(w, r)
				return
			}

			rules, err := privacy.NewRuleBuilder(settings, privacy.WithGlobalExceptions(nil)).Build()
			if err != nil {
				services.Logger.Error("failed to build rules", slog.String("error", err.Error()))
				next.ServeHTTP(w, r)
				return
			}

			ctx := server.WithRules(r.Context(), rules)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Verify interface compliance at compile time.
var _ server.MiddlewareRegistrar = (*ServiceRulesPlugin)(nil)
