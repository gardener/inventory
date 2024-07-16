// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"log/slog"
	"os"

	// "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

// Elb is the Elb client used by the workers to interact with AWS LoadBalancer API.
var Elb *elb.Client

func init() {
	// Trying to force go to run other source file first
	var err error
	cfg, err = config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("could not load AWS config", "err", err)
		os.Exit(1)
	}

	Elb = elb.NewFromConfig(cfg)
}
