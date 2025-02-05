// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import "errors"

// ErrNoProjectName is an error, which is returned when an expected Project Name
// was not specified as part of the task payload.
var ErrNoProjectName = errors.New("no project name specified")

// ErrNoProjectNamespace is an error, which is returned when an expected Project
// namespace was not specified as part of the task payload.
var ErrNoProjectNamespace = errors.New("no project namespace specified")

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
