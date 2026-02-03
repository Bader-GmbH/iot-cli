package api

import (
	"context"
)

// UsageData represents current usage metrics
type UsageData struct {
	YearMonth               string `json:"yearMonth"`
	BytesTransferred        int64  `json:"bytesTransferred"`
	BytesTransferredFormatted string `json:"bytesTransferredFormatted"`
}

// UsageHistoryData represents historical usage for a month
type UsageHistoryData struct {
	YearMonth               string `json:"yearMonth"`
	BytesTransferred        int64  `json:"bytesTransferred"`
	BytesTransferredFormatted string `json:"bytesTransferredFormatted"`
}

// GetUsage fetches current usage metrics
func (c *Client) GetUsage(ctx context.Context) (*UsageData, error) {
	var usage UsageData
	err := c.Get(ctx, "/api/usage", &usage)
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// GetUsageHistory fetches usage history
func (c *Client) GetUsageHistory(ctx context.Context) ([]UsageHistoryData, error) {
	var history []UsageHistoryData
	err := c.Get(ctx, "/api/usage/history", &history)
	if err != nil {
		return nil, err
	}
	return history, nil
}
