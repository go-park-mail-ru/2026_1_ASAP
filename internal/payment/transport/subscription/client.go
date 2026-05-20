package subscription

import (
	"context"

	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
)

type Client struct {
	api subscriptionv1.SubscriptionClient
}

func New(api subscriptionv1.SubscriptionClient) *Client {
	return &Client{api: api}
}

func (c *Client) Activate(ctx context.Context, userID int64, days int64) error {
	_, err := c.api.ActivateSubscription(ctx, &subscriptionv1.RequestActivateSubscription{
		UserId: userID,
		Days:   days,
	})
	return err
}
