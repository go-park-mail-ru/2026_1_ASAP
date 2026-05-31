package subscription

import (
	"context"

	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

type Client struct {
	api subscriptionv1.SubscriptionClient
}

func New(api subscriptionv1.SubscriptionClient) *Client {
	return &Client{api: api}
}

func (c *Client) IsActive(ctx context.Context, userID int64) (bool, error) {
	if c == nil || c.api == nil {
		return false, nil
	}
	resp, err := c.api.GetSubscription(ctx, &subscriptionv1.RequestGetSubscription{
		UserId: userID,
	})
	if err != nil {
		_, code, _ := grpcerr.Error(err)
		if code == int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_NOT_FOUND) ||
			code == int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_EXPIRED) {
			return false, nil
		}
		return false, err
	}
	return resp.GetActive(), nil
}
