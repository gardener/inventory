package clients

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var cfg aws.Config

// Ec2 is the EC2 client used by the workers to interact with AWS.
var Ec2 *ec2.Client

func init() {
	var err error
	cfg, err = config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("could not load AWS config", "err", err)
		os.Exit(1)
	}
	Ec2 = ec2.NewFromConfig(cfg)
}
