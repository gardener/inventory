// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package jwt provides an implementation of the JWT Auth Method for Vault.
package jwt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

// DefaultMountPath specifies the default mount path for the JWT
// Authentication Method.
const DefaultMountPath = "jwt"

// ErrNoToken is an error, which is returned when [Auth] is configured
// with an empty token.
var ErrNoToken = errors.New("no token specified")

// ErrInvalidMountPath is an error, which is returned when configuring [Auth]
// to use an invalid mount path for a Vault Authentication Method.
var ErrInvalidMountPath = errors.New("invalid auth method mount path specified")

// ErrNoRoleName is an error, which is returned when no role name was specified
// when creating a [Auth].
var ErrNoRoleName = errors.New("no role name specified")

// Auth implements support for the [JWT Authentication Method].
//
// [JWT Authentication Method]: https://developer.hashicorp.com/vault/docs/auth/jwt
type Auth struct {
	// roleName specifies the name of the role to use.
	roleName string

	// mountPath specifies the mount path for the JWT Authentication Method.
	mountPath string

	// token specifies the JWT token which will be used for authenticating
	// against the Vault Authentication Method endpoint.
	token string

	// tokenPath specifies a path from which to read the JWT token.
	tokenPath string
}

var _ vault.AuthMethod = &Auth{}

// Option is a function which configures [Auth].
type Option func(a *Auth) error

// New creates a new [Auth] and configures it with the given options.
//
// The default mount path for the JWT Authentication Method is
// [DefaultMountPath]. In order to configure a different mount path for the
// Authentication Method you can use the [WithMountPath] option.
//
// The JWT token which will be used for authentication against the Vault
// Authentication Method login endpoint may be specified either as a string,
// from path, or via an environment variable. In order to configure the token
// for authentication use the [WithToken], [WithTokenFromPath] or
// [WithTokenFromEnv] options.
func New(roleName string, opts ...Option) (*Auth, error) {
	if roleName == "" {
		return nil, ErrNoRoleName
	}

	auth := &Auth{
		roleName:  roleName,
		mountPath: DefaultMountPath,
	}

	for _, opt := range opts {
		if err := opt(auth); err != nil {
			return nil, err
		}
	}

	if auth.token == "" && auth.tokenPath == "" {
		return nil, ErrNoToken
	}

	if auth.mountPath == "" {
		return nil, ErrInvalidMountPath
	}

	return auth, nil
}

// Login implements the [vault.AuthMethod] interface.
func (a *Auth) Login(ctx context.Context, client *vault.Client) (*vault.Secret, error) {
	var token string

	switch {
	case a.token != "":
		token = a.token
	case a.tokenPath != "":
		data, err := os.ReadFile(filepath.Clean(a.tokenPath))
		if err != nil {
			return nil, err
		}
		token = string(data)
	}

	path := fmt.Sprintf("auth/%s/login", a.mountPath)
	data := map[string]any{
		"jwt":  strings.TrimSpace(token),
		"role": a.roleName,
	}

	return client.Logical().WriteWithContext(ctx, path, data)
}

// WithToken is an [Option], which configures [Auth] to use the given token
// when authenticating against the Vault JWT Authentication Method.
func WithToken(token string) Option {
	opt := func(a *Auth) error {
		a.token = token

		return nil
	}

	return opt
}

// WithTokenFromPath is an [Option], which configures [Auth] to read the
// token from the given path.
func WithTokenFromPath(path string) Option {
	opt := func(a *Auth) error {
		a.tokenPath = path

		return nil
	}

	return opt
}

// WithTokenFromEnv is an [Option], which configures [Auth] to read the token
// from the given environment variable.
func WithTokenFromEnv(env string) Option {
	opt := func(a *Auth) error {
		value := os.Getenv(env)
		a.token = value

		return nil
	}

	return opt
}

// WithMountPath is an [Option], which configures [Auth] to use the given
// mount path for the Vault Authentication Method.
func WithMountPath(mountPath string) Option {
	opt := func(a *Auth) error {
		a.mountPath = mountPath

		return nil
	}

	return opt
}
