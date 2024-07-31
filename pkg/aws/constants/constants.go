// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// PageSize represents the max number of items to fetch from the AWS API
	// during a paginated call.
	PageSize = 100

	// The value set for the type column of classic LBs.
	LoadBalancerClassicType  = "classic"
	LoadBalancerClassicState = "N/A"
)
