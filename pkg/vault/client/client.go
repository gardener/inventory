// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"log/slog"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// defaultReauthPeriod is an approximate percentage from the auth token TTL
// value after which re-authentication or token renew will be done.
const defaultReauthPeriod = 0.8

// ErrNoAuthMethod is an error, which is returned when attempting to login using
// an Auth Method, but no Auth Method implementation was configured.
var ErrNoAuthMethod = errors.New("no auth method implementation configured")

// ErrNoAuthInfo is an error, which is returned when a successful authentication
// to an Auth Method endpoint was performed, but no auth info was returned as
// part of the response.
var ErrNoAuthInfo = errors.New("no auth info returned")

// Option is a function which configures the [Client]
type Option func(c *Client) error

// Client is a wrapper around [vault.Client] with additional funtionality such
// as renewing authentication tokens.
type Client struct {
	*vault.Client

	config *vault.Config
	am     vault.AuthMethod
}

// ManageAuthTokenLifetime starts managing the auth token lifetime.
//
// It uses a periodic ticker, which will renew the auth token, if it is
// renewable. When the token is not renewable (e.g. batch tokens) a complete
// re-authentication will be done instead when ~ 80% of the token lifetime is
// reached.
func (c *Client) ManageAuthTokenLifetime(ctx context.Context) error {
	// First, get the auth token secret which we will be managing.
	var authInfo *vault.Secret
	var err error

	if c.am != nil {
		// If we are using an Auth Method, we need to login first.
		authInfo, err = c.login(ctx)
	} else {
		// Otherwise we can can simply lookup the configured token
		authInfo, err = c.lookupSelfToken(ctx)
	}

	if err != nil {
		return err
	}

	if authInfo == nil {
		return ErrNoAuthInfo
	}

	// Get token info
	ttl, err := authInfo.TokenTTL()
	if err != nil {
		return err
	}

	isRenewable, err := authInfo.TokenIsRenewable()
	if err != nil {
		return err
	}

	switch {
	case ttl <= 0:
		// Nothing to do here
		return nil
	case c.am == nil && !isRenewable:
		// We don't have an Auth Method implementation and the token is
		// not renewable. Nothing to do here as well.
		return nil
	case c.am != nil && !isRenewable:
		// We do have an Auth Method implementation and token is not
		// renewable.  Use a simple ticker and perform a complete
		// re-authentication when the token expiration approaches.
		duration := ttl.Seconds() * defaultReauthPeriod
		go c.reAuthPeriodically(ctx, time.Duration(duration)*time.Second)
	case isRenewable:
		// Token is renewable.
		duration := ttl.Seconds() * defaultReauthPeriod
		go c.renewPeriodically(ctx, time.Duration(duration)*time.Second)
	}

	return nil
}

// renewPeriodically attempts to renew the token periodically.
//
// If the token max TTL threshold has been reached it will perform a complete
// re-authentication.
func (c *Client) renewPeriodically(ctx context.Context, duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	var err error
	var authInfo *vault.Secret

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Attempt to renew first. If it fails, try to
			// re-authenticate if we use an Auth Method.
			slog.Info(
				"renewing vault token",
				"address", c.config.Address,
			)

			authInfo, err = c.Auth().Token().RenewSelfWithContext(ctx, 3600)
			if err != nil {
				slog.Error(
					"failed to renew vault token",
					"address", c.config.Address,
					"reason", err,
				)

				// Nothing to do when we don't have an Auth
				// Method implementation, so better luck next
				// time.
				if c.am == nil {
					continue
				}

				authInfo, err = c.login(ctx)
				if err != nil {
					slog.Error(
						"failed to authenticate with vault",
						"address", c.config.Address,
						"reason", err,
					)

					// Try again later
					continue
				}
			}

			if authInfo == nil {
				slog.Warn(
					"empty auth info returned from vault",
					"address", c.config.Address,
				)

				continue
			}

			// Read new auth token TTL and adjust the ticker accordingly
			ttl, err := authInfo.TokenTTL()
			if err != nil {
				slog.Warn(
					"cannot read vault auth token ttl",
					"address", c.config.Address,
					"reason", err,
				)

				continue
			}

			if ttl <= 0 {
				slog.Warn(
					"vault token ttl <= 0, will not attempt renewal",
					"address", c.config.Address,
				)

				return
			}

			newTickerDuration := ttl.Seconds() * defaultReauthPeriod
			ticker.Reset(time.Duration(newTickerDuration) * time.Second)
		}
	}
}

// reAuthPeriodically performs a complete re-authentication periodically.
func (c *Client) reAuthPeriodically(ctx context.Context, duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			authInfo, err := c.login(ctx)
			if err != nil {
				slog.Error(
					"failed to authenticate with vault",
					"address", c.config.Address,
					"reason", err,
				)

				continue
			}
			if authInfo == nil {
				slog.Warn(
					"empty auth info returned from vault",
					"address", c.config.Address,
				)

				continue
			}

			// Read new auth token TTL and adjust the ticker accordingly
			ttl, err := authInfo.TokenTTL()
			if err != nil {
				slog.Warn(
					"cannot read vault auth token ttl",
					"address", c.config.Address,
					"reason", err,
				)

				continue
			}

			if ttl <= 0 {
				slog.Warn(
					"vault token ttl <= 0, will not attempt re-authentication",
					"address", c.config.Address,
				)

				return
			}

			newTickerDuration := ttl.Seconds() * defaultReauthPeriod
			ticker.Reset(time.Duration(newTickerDuration) * time.Second)
		}
	}
}

// login performs a login using the configured Auth Method implementation.
func (c *Client) login(ctx context.Context) (*vault.Secret, error) {
	slog.Info(
		"authenticating with vault",
		"address", c.config.Address,
	)

	if c.am == nil {
		return nil, ErrNoAuthMethod
	}

	authInfo, err := c.Auth().Login(ctx, c.am)
	if err != nil {
		return nil, err
	}

	if authInfo == nil {
		return nil, ErrNoAuthInfo
	}

	return authInfo, nil
}

// lookupSelfToken gets information about the locally authenticated token.
func (c *Client) lookupSelfToken(ctx context.Context) (*vault.Secret, error) {
	return c.Auth().Token().LookupSelfWithContext(ctx)
}

// New creates a new [Client] from the given config and options.
func New(config *vault.Config, opts ...Option) (*Client, error) {
	vaultClient, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Client: vaultClient,
		config: config,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// WithAuthMethod is an [Option], which configures the [Client] to use the given
// Auth Method.
func WithAuthMethod(am vault.AuthMethod) Option {
	opt := func(c *Client) error {
		c.am = am

		return nil
	}

	return opt
}
