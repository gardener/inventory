// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import "errors"

// ErrNoVirtualGardenClientFound is an error, which is returned when no Virtual
// Garden client was found.
var ErrNoVirtualGardenClientFound = errors.New("no virtual garden client found")

// ErrNoSeedCluster is an error, which is returned when an expected Seed Cluster
// was not specified.
var ErrNoSeedCluster = errors.New("no seed cluster specified")

// ErrMissingProviderConfig is returned when an expected provider config is
// missing from the payload.
var ErrNoProviderConfig = errors.New("no provider config specified")

// ErrMissingCloudProfileName is returned when an expected cloud profile name is
// missing from the payload.
var ErrNoCloudProfileName = errors.New("no cloud profile name specified")

// ErrNoPayload is an error, which is returned by task handlers, which expect
// payload, but none was provided.
var ErrNoPayload = errors.New("no payload specified")
