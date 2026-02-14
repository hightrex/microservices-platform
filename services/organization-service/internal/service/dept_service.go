package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// DeptService handles department business logic.
type DeptService struct {
	depts DeptRepository
}

// NewDeptService creates a new DeptService.
func NewDeptService(depts DeptRepository) *DeptService {
	return &DeptService{depts: depts}
}

// Create creates a new department within an organization.
func (s *DeptService) Create(ctx context.Context, orgID uuid.UUID, req models.CreateDeptRequest) (*models.DeptResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	dept := &models.Department{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	if err := s.depts.Create(ctx, orgID, dept); err != nil {
		return nil, err
	}

	resp := dept.ToResponse()
	return &resp, nil
}

// List retrieves all departments for an organization, building a hierarchy.
func (s *DeptService) List(ctx context.Context, orgID uuid.UUID) ([]models.DeptResponse, error) {
	depts, err := s.depts.List(ctx, orgID)
	if err != nil {
		return nil, err
	}

	return buildDeptHierarchy(depts), nil
}

// Update performs a partial update on a department.
func (s *DeptService) Update(ctx context.Context, orgID, deptID uuid.UUID, req models.UpdateDeptRequest) (*models.DeptResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	fields := make(map[string]interface{})
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.ParentID != nil {
		fields["parent_id"] = *req.ParentID
	}

	if len(fields) == 0 {
		return nil, errors.BadRequest("No fields to update", nil)
	}

	if err := s.depts.Update(ctx, orgID, deptID, fields); err != nil {
		return nil, err
	}

	dept, err := s.depts.GetByID(ctx, orgID, deptID)
	if err != nil {
		return nil, err
	}

	resp := dept.ToResponse()
	return &resp, nil
}

// Delete removes a department.
func (s *DeptService) Delete(ctx context.Context, orgID, deptID uuid.UUID) error {
	return s.depts.Delete(ctx, orgID, deptID)
}

// buildDeptHierarchy assembles a flat list of departments into a tree structure.
func buildDeptHierarchy(depts []models.Department) []models.DeptResponse {
	deptMap := make(map[uuid.UUID]*models.DeptResponse)
	var roots []models.DeptResponse

	// First pass: create response objects
	for _, d := range depts {
		resp := d.ToResponse()
		resp.Children = make([]models.DeptResponse, 0)
		deptMap[d.ID] = &resp
	}

	// Second pass: build hierarchy
	for _, d := range depts {
		resp := deptMap[d.ID]
		if d.ParentID != nil {
			if parent, ok := deptMap[*d.ParentID]; ok {
				parent.Children = append(parent.Children, *resp)
				continue
			}
		}
		roots = append(roots, *resp)
	}

	if roots == nil {
		roots = make([]models.DeptResponse, 0)
	}
	return roots
}
