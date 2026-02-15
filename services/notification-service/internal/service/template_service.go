package service

import (
	"bytes"
	"context"
	"html/template"
	"strings"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// TemplateService handles notification template business logic.
type TemplateService struct {
	templateRepo TemplateRepository
	cache        TemplateCache
	publisher    EventPublisher
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(repo TemplateRepository, cache TemplateCache, publisher EventPublisher) *TemplateService {
	return &TemplateService{
		templateRepo: repo,
		cache:        cache,
		publisher:    publisher,
	}
}

// CreateTemplate validates and creates a new notification template.
func (s *TemplateService) CreateTemplate(ctx context.Context, req models.CreateTemplateRequest, createdBy uuid.UUID) (*models.TemplateResponse, error) {
	// Validate template syntax
	if err := validateTemplateSyntax(req.BodyTemplate); err != nil {
		return nil, errors.BadRequest("Invalid body template syntax: "+err.Error(), err)
	}
	if req.SubjectTemplate != nil {
		if err := validateTemplateSyntax(*req.SubjectTemplate); err != nil {
			return nil, errors.BadRequest("Invalid subject template syntax: "+err.Error(), err)
		}
	}

	language := req.Language
	if language == "" {
		language = "en"
	}

	tmpl := &models.NotificationTemplate{
		Name:               req.Name,
		Channel:            req.Channel,
		SubjectTemplate:    req.SubjectTemplate,
		BodyTemplate:       req.BodyTemplate,
		TemplateDataSchema: req.TemplateDataSchema,
		Language:           language,
		IsActive:           true,
		CreatedBy:          &createdBy,
	}

	if err := s.templateRepo.Create(ctx, tmpl); err != nil {
		return nil, err
	}

	// Publish event
	if err := s.publisher.Publish(ctx, "notification-events", "notification.template_created", map[string]interface{}{
		"template_id": tmpl.ID,
		"name":        tmpl.Name,
		"channel":     tmpl.Channel,
	}); err != nil {
		logger.Warn().Err(err).Str("template_id", tmpl.ID.String()).Msg("Failed to publish template_created event")
	}

	resp := tmpl.ToResponse()
	return &resp, nil
}

// GetTemplate retrieves a template by ID, with caching.
func (s *TemplateService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.TemplateResponse, error) {
	// Try cache first
	cached, err := s.cache.GetTemplate(ctx, id)
	if err == nil && cached != nil {
		resp := cached.ToResponse()
		return &resp, nil
	}

	tmpl, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache
	if cacheErr := s.cache.SetTemplate(ctx, tmpl); cacheErr != nil {
		logger.Warn().Err(cacheErr).Str("template_id", id.String()).Msg("Failed to cache template")
	}

	resp := tmpl.ToResponse()
	return &resp, nil
}

// ListTemplates retrieves templates with filtering, scoped to tenant.
func (s *TemplateService) ListTemplates(ctx context.Context, filter models.TemplateFilter, page models.Pagination) ([]models.TemplateResponse, int, error) {
	templates, total, err := s.templateRepo.List(ctx, filter, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = t.ToResponse()
	}
	return responses, total, nil
}

// UpdateTemplate performs a partial update on a template and invalidates cache.
func (s *TemplateService) UpdateTemplate(ctx context.Context, id uuid.UUID, req models.UpdateTemplateRequest) error {
	fields := make(map[string]interface{})

	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.SubjectTemplate != nil {
		if err := validateTemplateSyntax(*req.SubjectTemplate); err != nil {
			return errors.BadRequest("Invalid subject template syntax: "+err.Error(), err)
		}
		fields["subject_template"] = *req.SubjectTemplate
	}
	if req.BodyTemplate != nil {
		if err := validateTemplateSyntax(*req.BodyTemplate); err != nil {
			return errors.BadRequest("Invalid body template syntax: "+err.Error(), err)
		}
		fields["body_template"] = *req.BodyTemplate
	}
	if req.TemplateDataSchema != nil {
		fields["template_data_schema"] = req.TemplateDataSchema
	}
	if req.Language != nil {
		fields["language"] = *req.Language
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if err := s.templateRepo.Update(ctx, id, fields); err != nil {
		return err
	}

	// Invalidate cache
	if err := s.cache.InvalidateTemplate(ctx, id); err != nil {
		logger.Warn().Err(err).Str("template_id", id.String()).Msg("Failed to invalidate template cache")
	}

	return nil
}

// DeleteTemplate soft-deletes a template (sets is_active=false) and invalidates cache.
func (s *TemplateService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if err := s.templateRepo.Delete(ctx, id); err != nil {
		return err
	}

	if err := s.cache.InvalidateTemplate(ctx, id); err != nil {
		logger.Warn().Err(err).Str("template_id", id.String()).Msg("Failed to invalidate template cache on delete")
	}

	return nil
}

// RenderTemplate renders a template with the given data using Go's html/template package.
// It uses a restricted function map that prevents dangerous operations.
func (s *TemplateService) RenderTemplate(tmpl *models.NotificationTemplate, data map[string]interface{}) (subject string, body string, err error) {
	// Render body
	body, err = renderSafe(tmpl.BodyTemplate, data)
	if err != nil {
		return "", "", errors.BadRequest("Failed to render body template: "+err.Error(), err)
	}

	// Render subject if present
	if tmpl.SubjectTemplate != nil && *tmpl.SubjectTemplate != "" {
		subject, err = renderSafe(*tmpl.SubjectTemplate, data)
		if err != nil {
			return "", "", errors.BadRequest("Failed to render subject template: "+err.Error(), err)
		}
	}

	return subject, body, nil
}

// validateTemplateSyntax checks that a template string is valid Go template syntax.
func validateTemplateSyntax(tmplStr string) error {
	_, err := template.New("validation").Funcs(safeFuncMap()).Parse(tmplStr)
	return err
}

// renderSafe renders a Go template string with a restricted function map to prevent injection.
func renderSafe(tmplStr string, data map[string]interface{}) (string, error) {
	t, err := template.New("render").Funcs(safeFuncMap()).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// safeFuncMap returns a template function map with only safe, approved functions.
func safeFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title, //nolint:staticcheck // Title is fine for templates
		"trim":  strings.TrimSpace,
	}
}
