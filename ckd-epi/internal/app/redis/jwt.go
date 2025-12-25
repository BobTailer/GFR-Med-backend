package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const jwtPrefix = "jwt."
const sessionPrefix = "session."

func getJWTKey(token string) string {
	return servicePrefix + jwtPrefix + token
}

func getSessionKey(token string) string {
	return servicePrefix + sessionPrefix + token
}

func (c *Client) WriteJWTToBlacklist(ctx context.Context, jwtStr string, jwtTTL time.Duration) error {
	key := getJWTKey(jwtStr)
	return c.client.Set(ctx, key, true, jwtTTL).Err()
}

func (c *Client) StoreSession(ctx context.Context, token string, userID uint, ttl time.Duration) error {
	key := getSessionKey(token)
	return c.client.Set(ctx, key, userID, ttl).Err()
}

func (c *Client) GetSessionUserID(ctx context.Context, token string) (uint, error) {
	key := getSessionKey(token)
	result := c.client.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, fmt.Errorf("session not found")
	}
	if result.Err() != nil {
		return 0, result.Err()
	}
	userID, err := result.Uint64()
	if err != nil {
		return 0, err
	}
	return uint(userID), nil
}

func (c *Client) DeleteSession(ctx context.Context, token string) error {
	key := getSessionKey(token)
	return c.client.Del(ctx, key).Err()
}

func (c *Client) CheckJWTInBlacklist(ctx context.Context, jwtStr string) error {
	key := getJWTKey(jwtStr)
	err := c.client.Get(ctx, key).Err()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("token is in blacklist")
}

