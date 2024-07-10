// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
//
// Package tokenfile implements utilities for retrieving identity tokens from a
// given path.
//
// The token retriever is meant to be plugged-in to an AWS Web Identity
// Credentials Provider, so that short-lived JWT tokens can be exchanged for
// temporary security credentials when accessing AWS resources.

package tokenfile

import (
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
)

const (
	// TokenRetrieverName specifies the name of the Token Retriever.
	TokenRetrieverName = "token_file"
)

// ErrNoTokenPath is an error, which is returned, when the [TokenRetriever] is
// configured without a path to the token file.
var ErrNoTokenPath = errors.New("no token path specified")

// TokenRetriever retrieves an identity token from a given path.
//
// TokenRetriever implements the [stscreds.IdentityTokenRetriever] interface.
type TokenRetriever struct {
	// path specifies the path to the identity token file.
	path string
}

var _ stscreds.IdentityTokenRetriever = &TokenRetriever{}

// GetIdentityToken implements the [stscreds.IdentityTokenRetriever] interface.
func (t *TokenRetriever) GetIdentityToken() ([]byte, error) {
	return os.ReadFile(t.path)
}

// Option is a function which configures a [TokenRetriever] instance.
type Option func(*TokenRetriever)

// NewTokenRetriever creates a new [TokenRetriever] and configures it using the
// provided options.
func NewTokenRetriever(opts ...Option) (*TokenRetriever, error) {
	tokenRetriever := &TokenRetriever{}
	for _, opt := range opts {
		opt(tokenRetriever)
	}

	if tokenRetriever.path == "" {
		return nil, ErrNoTokenPath
	}

	return tokenRetriever, nil
}

// WithPath returns an [Option], which configures the [TokenRetriever] to use
// the given filepath to read the contents of the identity token.
func WithPath(path string) Option {
	opt := func(t *TokenRetriever) {
		t.path = path
	}

	return opt
}
