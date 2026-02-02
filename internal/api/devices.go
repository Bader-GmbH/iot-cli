package api

import (
	"context"

	"github.com/Bader-GmbH/iot-cli/pkg/models"
)

// ListDevices retrieves all devices
func (c *Client) ListDevices(ctx context.Context) ([]models.Device, error) {
	var devices []models.Device
	if err := c.Get(ctx, "/api/devices", &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// GetDevice retrieves a single device by ID
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*models.Device, error) {
	var device models.Device
	if err := c.Get(ctx, "/api/devices/"+deviceID, &device); err != nil {
		return nil, err
	}
	return &device, nil
}

// ListPendingDevices retrieves devices waiting for approval
func (c *Client) ListPendingDevices(ctx context.Context) ([]models.Device, error) {
	var devices []models.Device
	if err := c.Get(ctx, "/api/devices/pending", &devices); err != nil {
		return nil, err
	}
	return devices, nil
}