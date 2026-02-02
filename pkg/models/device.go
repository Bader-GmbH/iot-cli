package models

import (
	"fmt"
	"time"
)

// DeviceStatus represents the approval status of a device
type DeviceStatus string

const (
	DeviceStatusPending        DeviceStatus = "PENDING"
	DeviceStatusApproved       DeviceStatus = "APPROVED"
	DeviceStatusRejected       DeviceStatus = "REJECTED"
	DeviceStatusDecommissioned DeviceStatus = "DECOMMISSIONED"
)

// Device represents an IoT device
type Device struct {
	ID                  string       `json:"id"`
	TenantID            string       `json:"tenantId"`
	Name                string       `json:"name"`
	Online              bool         `json:"online"`
	LastHeartbeat       int64        `json:"lastHeartbeat"`
	Status              DeviceStatus `json:"status"`
	GroupID             *string      `json:"groupId,omitempty"`
	GroupName           *string      `json:"groupName,omitempty"`
	RegistrationTokenID *string      `json:"registrationTokenId,omitempty"`
	ApprovedAt          *time.Time   `json:"approvedAt,omitempty"`
	ApprovedBy          *string      `json:"approvedBy,omitempty"`
	RejectedAt          *time.Time   `json:"rejectedAt,omitempty"`
	DecommissionedAt    *time.Time   `json:"decommissionedAt,omitempty"`
}

// LastSeenString returns a human-readable string for when the device was last seen
func (d *Device) LastSeenString() string {
	if d.LastHeartbeat == 0 {
		return "never"
	}

	lastSeen := time.UnixMilli(d.LastHeartbeat)
	duration := time.Since(lastSeen)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return formatDuration(mins, "minute")
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return formatDuration(hours, "hour")
	default:
		days := int(duration.Hours() / 24)
		return formatDuration(days, "day")
	}
}

func formatDuration(n int, unit string) string {
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}

// OnlineStatus returns a string representation of the online status
func (d *Device) OnlineStatus() string {
	if d.Online {
		return "online"
	}
	return "offline"
}