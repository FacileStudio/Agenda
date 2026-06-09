package settings

import (
	"context"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (c *Controller) getSettings(ctx context.Context) (*SettingsResponse, error) {
	s, err := c.service.getSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &SettingsResponse{Settings: *s}, nil
}

func (c *Controller) updateSettings(ctx context.Context, req *UpdateRequest) (*SettingsResponse, error) {
	s, err := c.service.updateSettings(ctx, req)
	if err != nil {
		return nil, err
	}
	return &SettingsResponse{Settings: *s}, nil
}
