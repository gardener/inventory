// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hibiken/asynq"
	"github.com/olekukonko/tablewriter"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gardener/inventory/internal/pkg/migrations"
	"github.com/gardener/inventory/pkg/aws/stscreds/kubesatoken"
	"github.com/gardener/inventory/pkg/aws/stscreds/provider"
	"github.com/gardener/inventory/pkg/aws/stscreds/tokenfile"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/core/config"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	stringutils "github.com/gardener/inventory/pkg/utils/strings"
)

// na is the const used to represent N/A values
const na = "N/A"

// configKey is the key used to store the parsed configuration in the context
type configKey struct{}

// errInvalidDSN error is returned, if the DSN configuration is incorrect, or
// empty.
var errInvalidDSN = errors.New("invalid or missing database configuration")

// errInvalidWorkerConcurrency error is returned when the worker concurrency
// setting is invalid, e.g. it is <= 0.
var errInvalidWorkerConcurrency = errors.New("invalid worker concurrency")

// errInvalidRedisEndpoint is returned when Redis is configured with an invalid
// endpoint.
var errInvalidRedisEndpoint = errors.New("invalid or missing redis endpoint")

// errNoDashboardAddress is an error, which is returned when the Dashboard
// service was not configured with a bind address.
var errNoDashboardAddress = errors.New("no bind address specified")

// errNoAWSRegion is an error which is returned when there was no region or
// default region configured for the AWS client.
var errNoAWSRegion = errors.New("no AWS region specified")

// errNoAWSTokenRetriever is an error, which is returned when there was no token
// retriever name specified.
var errNoAWSTokenRetriever = errors.New("no AWS token retriever specified")

// errUnknownAWSTokenRetriever is an error, which is returned when using an
// unknown/unsupported identity token retriever.
var errUnknownAWSTokenRetriever = errors.New("unknown AWS token retriever specified")

// errNoAWSServiceCredentials is an error, which is returned when an AWS service
// (e.g. EC2, ELBv2, etc.) does not have any named credentials configured.
var errNoAWSServiceCredentials = errors.New("no credentials specified for service")

// errUnknownAWSNamedCredentials is an error which is returned when a service is
// using an unknown AWS named credentials.
var errUnknownAWSNamedCredentials = errors.New("unknown AWS named credentials")

// errInvalidLogLevel is an error, which is returned when an invalid log level
// has been configured.
var errInvalidLogLevel = errors.New("invalid log level")

// errInvalidLogFormat is an error, which is returned when an invalid log format
// has been configured.
var errInvalidLogFormat = errors.New("invalid log format")

// getConfig extracts and returns the [config.Config] from app's context.
func getConfig(ctx *cli.Context) *config.Config {
	conf := ctx.Context.Value(configKey{}).(*config.Config)
	return conf
}

// validateDBConfig validates the database configuration settings.
func validateDBConfig(conf *config.Config) error {
	if conf.Database.DSN == "" {
		return errInvalidDSN
	}

	return nil
}

// validateAWSConfig validates the AWS configuration settings.
func validateAWSConfig(conf *config.Config) error {
	// Region or default region must be specified
	if conf.AWS.Region == "" && conf.AWS.DefaultRegion == "" {
		return errNoAWSRegion
	}

	// Make sure that services have configured named credentials
	services := map[string][]string{
		"ec2":   conf.AWS.Services.EC2.UseCredentials,
		"elb":   conf.AWS.Services.ELB.UseCredentials,
		"elbv2": conf.AWS.Services.ELBv2.UseCredentials,
		"s3":    conf.AWS.Services.S3.UseCredentials,
	}

	for service, namedCredentials := range services {
		// We expect at least one named credential to be present per
		// service
		if len(namedCredentials) == 0 {
			return fmt.Errorf("%w: %s", errNoAWSServiceCredentials, service)
		}

		// Validate that the named credentials used by the services are
		// actually configured.
		for _, nc := range namedCredentials {
			if _, ok := conf.AWS.Credentials[nc]; !ok {
				return fmt.Errorf("%w: service %s refers %s", errUnknownAWSNamedCredentials, service, nc)
			}
		}
	}

	// Each named credential must use a valid token retriever
	supportedTokenRetrievers := []string{
		config.DefaultAWSTokenRetriever,
		kubesatoken.TokenRetrieverName,
		tokenfile.TokenRetrieverName,
	}
	for name, creds := range conf.AWS.Credentials {
		if creds.TokenRetriever == "" {
			return fmt.Errorf("%w: %s", errNoAWSTokenRetriever, name)
		}

		if !slices.Contains(supportedTokenRetrievers, creds.TokenRetriever) {
			return fmt.Errorf("%w: %s", errUnknownAWSTokenRetriever, creds.TokenRetriever)
		}
	}

	return nil
}

// validateWorkerConfig validates the worker configuration settings.
func validateWorkerConfig(conf *config.Config) error {
	if conf.Worker.Concurrency <= 0 {
		return fmt.Errorf("%w: %d", errInvalidWorkerConcurrency, conf.Worker.Concurrency)
	}

	return nil
}

// validateDashboardConfig validates the Dashboard service configuration.
func validateDashboardConfig(conf *config.Config) error {
	if conf.Dashboard.Address == "" {
		return errNoDashboardAddress
	}

	_, err := url.Parse(conf.Dashboard.PrometheusEndpoint)
	return err
}

// validateRedisConfig validates the Redis configuration settings.
func validateRedisConfig(conf *config.Config) error {
	if conf.Redis.Endpoint == "" {
		return errInvalidRedisEndpoint
	}

	return nil
}

// logLevel represents the log level
type logLevel string

var (
	// The supported log levels
	levelInfo  logLevel = "info"
	levelWarn  logLevel = "warn"
	levelError logLevel = "error"
	levelDebug logLevel = "debug"
)

// logFormat represents the format of log events
type logFormat string

var (
	// The supported log formats
	logFormatText logFormat = "text"
	logFormatJSON logFormat = "json"
)

// newLogger creates a new [slog.Logger] based on the provided [config.Config]
// spec, which outputs to the given [io.Writer].
func newLogger(w io.Writer, conf *config.Config) (*slog.Logger, error) {
	// Defaults, if we don't have any logging settings
	if conf.Logging.Level == "" {
		conf.Logging.Level = string(levelInfo)
	}

	if conf.Logging.Format == "" {
		conf.Logging.Format = string(logFormatText)
	}

	// Supported log levels
	levels := map[logLevel]slog.Level{
		levelInfo:  slog.LevelInfo,
		levelWarn:  slog.LevelWarn,
		levelError: slog.LevelError,
		levelDebug: slog.LevelDebug,
	}

	level, ok := levels[logLevel(conf.Logging.Level)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errInvalidLogLevel, string(conf.Logging.Level))
	}

	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		AddSource: conf.Logging.AddSource,
		Level:     level,
	}

	switch logFormat(conf.Logging.Format) {
	case logFormatText:
		handler = slog.NewTextHandler(w, handlerOpts)
	case logFormatJSON:
		handler = slog.NewJSONHandler(w, handlerOpts)
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidLogFormat, conf.Logging.Format)
	}

	// Add default attributes to the logger
	attrs := make([]slog.Attr, 0)
	for k, v := range conf.Logging.Attributes {
		attrs = append(attrs, slog.Any(k, v))
	}
	logger := slog.New(handler.WithAttrs(attrs))

	return logger, nil
}

// newAWSSTSClient creates a new [sts.Client] based on the provided
// [config.Config] spec.
func newAWSSTSClient(conf *config.Config) *sts.Client {
	awsConf := aws.Config{
		Region: conf.AWS.Region,
		AppID:  conf.AWS.AppID,
	}
	client := sts.NewFromConfig(awsConf)

	return client
}

// newKubeSATokenCredentialsProvider creates a new [aws.CredentialsProvider],
// which uses Kubernetes Service Account Tokens for exchanging them with
// temporary security credentials when accessing AWS resources.  The token
// retriever is configured based on the provided [config.AWSCredentialsConfig].
func newKubeSATokenCredentialsProvider(conf *config.Config, creds config.AWSCredentialsConfig) (aws.CredentialsProvider, error) {
	tokenRetriever, err := kubesatoken.NewTokenRetriever(
		kubesatoken.WithKubeconfig(creds.KubeSATokenRetriever.Kubeconfig),
		kubesatoken.WithServiceAccount(creds.KubeSATokenRetriever.ServiceAccount),
		kubesatoken.WithNamespace(creds.KubeSATokenRetriever.Namespace),
		kubesatoken.WithAudiences(creds.KubeSATokenRetriever.Audiences),
		kubesatoken.WithTokenExpiration(creds.KubeSATokenRetriever.Duration),
	)

	if err != nil {
		return nil, err
	}

	providerSpec := &provider.Spec{
		Client:          newAWSSTSClient(conf),
		RoleARN:         creds.KubeSATokenRetriever.RoleARN,
		RoleSessionName: creds.KubeSATokenRetriever.RoleSessionName,
		Duration:        creds.KubeSATokenRetriever.Duration,
		TokenRetriever:  tokenRetriever,
	}

	return provider.New(providerSpec)
}

// newTokenFileCredentialsProvider creates a new [aws.CredentialsProvider],
// which reads a JWT token from a specified path and exchanges the token for
// temporary security credentials when accessing AWS resources.  The token
// retriever is configured based on the provided [config.AWSCredentialsConfig].
func newTokenFileCredentialsProvider(conf *config.Config, creds config.AWSCredentialsConfig) (aws.CredentialsProvider, error) {
	tokenRetriever, err := tokenfile.NewTokenRetriever(
		tokenfile.WithPath(creds.TokenFileRetriever.Path),
	)

	if err != nil {
		return nil, err
	}

	providerSpec := &provider.Spec{
		Client:          newAWSSTSClient(conf),
		RoleARN:         creds.TokenFileRetriever.RoleARN,
		RoleSessionName: creds.TokenFileRetriever.RoleSessionName,
		Duration:        creds.TokenFileRetriever.Duration,
		TokenRetriever:  tokenRetriever,
	}

	return provider.New(providerSpec)
}

// loadAWSConfig loads the AWS configurations for the given named credentials.
func loadAWSConfig(ctx context.Context, conf *config.Config, namedCredentials string) (aws.Config, error) {
	creds, ok := conf.AWS.Credentials[namedCredentials]
	if !ok {
		return aws.Config{}, fmt.Errorf("%w: %s", errUnknownAWSNamedCredentials, namedCredentials)
	}

	// Default set of options
	opts := []func(o *awsconfig.LoadOptions) error{
		awsconfig.WithRegion(conf.AWS.Region),
		awsconfig.WithDefaultRegion(conf.AWS.DefaultRegion),
		awsconfig.WithAppID(conf.AWS.AppID),
	}

	switch creds.TokenRetriever {
	case config.DefaultAWSTokenRetriever:
		// Load shared credentials config only
		break
	case kubesatoken.TokenRetrieverName:
		credsProvider, err := newKubeSATokenCredentialsProvider(conf, creds)
		if err != nil {
			return aws.Config{}, err
		}
		opts = append(opts, awsconfig.WithCredentialsProvider(credsProvider))
	case tokenfile.TokenRetrieverName:
		credsProvider, err := newTokenFileCredentialsProvider(conf, creds)
		if err != nil {
			return aws.Config{}, err
		}
		opts = append(opts, awsconfig.WithCredentialsProvider(credsProvider))
	default:
		return aws.Config{}, errUnknownAWSTokenRetriever
	}

	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// configureEC2Clientset configures the [awsclients.EC2Clientset] registry.
func configureEC2Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.EC2.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := ec2.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*ec2.Client]{
			NamedCredentials: namedCreds,
			AccountID:        stringutils.StringFromPointer(callerIdentity.Account),
			ARN:              stringutils.StringFromPointer(callerIdentity.Arn),
			UserID:           stringutils.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.EC2Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "ec2",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureELBClientset configures the [awsclients.ELBClientset] registry.
func configureELBClientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.ELB.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := elb.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*elb.Client]{
			NamedCredentials: namedCreds,
			AccountID:        stringutils.StringFromPointer(callerIdentity.Account),
			ARN:              stringutils.StringFromPointer(callerIdentity.Arn),
			UserID:           stringutils.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.ELBClientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "elb",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureELBv2Clientset configures the [awsclients.ELBv2Clientset] registry.
func configureELBv2Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.ELBv2.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := elbv2.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*elbv2.Client]{
			NamedCredentials: namedCreds,
			AccountID:        stringutils.StringFromPointer(callerIdentity.Account),
			ARN:              stringutils.StringFromPointer(callerIdentity.Arn),
			UserID:           stringutils.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.ELBv2Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "elbv2",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureS3Clientset configures the [awsclients.S3Clientset] registry.
func configureS3Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.S3.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := s3.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*s3.Client]{
			NamedCredentials: namedCreds,
			AccountID:        stringutils.StringFromPointer(callerIdentity.Account),
			ARN:              stringutils.StringFromPointer(callerIdentity.Arn),
			UserID:           stringutils.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.S3Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "s3",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureAWSClients creates the AWS clients for the supported by Inventory
// AWS services and registers them.
func configureAWSClients(ctx context.Context, conf *config.Config) error {
	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"ec2":   configureEC2Clientset,
		"elb":   configureELBClientset,
		"elbv2": configureELBv2Clientset,
		"s3":    configureS3Clientset,
	}

	for svc, configFunc := range configFuncs {
		if err := configFunc(ctx, conf); err != nil {
			return fmt.Errorf("unable to configure AWS clients for %s: %w", svc, err)
		}
	}

	return nil
}

// newRedisClientOpt returns a new [asynq.RedisClientOpt] from the given config.
func newRedisClientOpt(conf *config.Config) asynq.RedisClientOpt {
	// TODO: Handle authentication, TLS, etc.
	opts := asynq.RedisClientOpt{
		Addr: conf.Redis.Endpoint,
	}

	return opts
}

// newClient creates a new [asynq.Client] from the given config
func newClient(conf *config.Config) *asynq.Client {
	redisClientOpt := newRedisClientOpt(conf)
	return asynq.NewClient(redisClientOpt)
}

// newInspector returns a new [asynq.Inspector] from the given config.
func newInspector(conf *config.Config) *asynq.Inspector {
	redisClientOpt := newRedisClientOpt(conf)
	return asynq.NewInspector(redisClientOpt)
}

// newServer creates a new [asynq.Server] from the given config.
func newServer(conf *config.Config) *asynq.Server {
	redisClientOpt := newRedisClientOpt(conf)

	// TODO: Logger, priority queues, etc.
	logLevel := asynq.InfoLevel
	if conf.Debug {
		logLevel = asynq.DebugLevel
	}

	config := asynq.Config{
		Concurrency:  conf.Worker.Concurrency,
		LogLevel:     logLevel,
		ErrorHandler: asynqutils.NewDefaultErrorHandler(),
	}

	server := asynq.NewServer(redisClientOpt, config)

	return server
}

// newDB returns a new [bun.DB] database from the given config.
func newDB(conf *config.Config) *bun.DB {
	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(conf.Database.DSN)))
	db := bun.NewDB(pgdb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(conf.Debug)))

	return db
}

// newMigrator creates a new [github.com/uptrace/bun/migrate.Migrator] from the
// given config.
func newMigrator(conf *config.Config, db *bun.DB) (*migrate.Migrator, error) {
	// By default we will use the bundled migrations, unless we have an
	// explicitely specified alternate migrations directory.
	m := migrations.Migrations
	migrationDir := conf.Database.MigrationDirectory

	if migrationDir != "" {
		m = migrate.NewMigrations(migrate.WithMigrationsDirectory(migrationDir))
		err := m.Discover(os.DirFS(migrationDir))
		switch {
		case err == nil:
			break
		case errors.Is(err, fs.ErrNotExist):
			slog.Warn(
				"falling back to bundled migrations",
				"reason", "migration path does not exist",
				"path", migrationDir,
			)
			m = migrations.Migrations
		default:
			// Any other error should bubble up to the caller
			return nil, fmt.Errorf("failed to discover migrations from %s: %w", migrationDir, err)
		}
	}

	migrator := migrate.NewMigrator(db, m, migrate.WithMarkAppliedOnSuccess(true))
	return migrator, nil
}

// newScheduler creates a new [asynq.Scheduler] from the given config.
func newScheduler(conf *config.Config) *asynq.Scheduler {
	redisClientOpt := newRedisClientOpt(conf)

	// TODO: Logger, etc.
	// TODO: PostEnqueue hook to emit metrics per tasks
	preEnqueueFunc := func(t *asynq.Task, opts []asynq.Option) {
		slog.Info("enqueueing task", "name", t.Type())
	}

	errEnqueueFunc := func(t *asynq.Task, opts []asynq.Option, err error) {
		slog.Error("failed to enqueue", "name", t.Type(), "error", err)
	}

	logLevel := asynq.InfoLevel
	if conf.Debug {
		logLevel = asynq.DebugLevel
	}

	opts := &asynq.SchedulerOpts{
		PreEnqueueFunc:      preEnqueueFunc,
		EnqueueErrorHandler: errEnqueueFunc,
		LogLevel:            logLevel,
	}

	scheduler := asynq.NewScheduler(redisClientOpt, opts)
	return scheduler
}

// newTableWriter creates a new [tablewriter.Table] with the given [io.Writer]
// and headers
func newTableWriter(w io.Writer, headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(w)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetBorder(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	return table
}

func newGardenConfigs(conf *config.Config) (map[string]*rest.Config, error) {

	// 1. Check for token according the configuration
	if conf.VirtualGarden.TokenPath != "" {
		return constructGardenConfigWithToken(conf)
	}

	// 2. Check for kubeconfig in the configuration or an env variable
	// Attempt to read the kubeconfig from the configuration file
	configs := make(map[string]*rest.Config)
	kubeconfig := virtualGardenKubeconfig(conf)
	if kubeconfig != "" {
		// Add any additional contexts from the kubeconfig, if present
		apiConfig, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
		for name := range apiConfig.Contexts {
			contextName := fetchContextName(name, conf.VirtualGarden.Environment)
			clientConfig := clientcmd.NewNonInteractiveClientConfig(*apiConfig, name, &clientcmd.ConfigOverrides{}, nil)
			restConfig, err := clientConfig.ClientConfig()
			if err != nil {
				slog.Error("failed to create rest config, skipping", "context", contextName, "err", err)
				continue
			}
			configs[contextName] = restConfig
		}
		if _, found := configs[gardenerclient.VIRTUAL_GARDEN]; !found {
			return nil, fmt.Errorf("no context found for the virtual garden in the kubeconfig")
		}
		return configs, nil
	}

	// If there is no token and the kubeconfig is not set, we are running in a testing environment
	// 3. Check for in-cluster config - for testing purposes
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	configs[gardenerclient.VIRTUAL_GARDEN] = inClusterConfig
	return configs, nil

}

func constructGardenConfigWithToken(conf *config.Config) (map[string]*rest.Config, error) {
	// Check if the token file exists
	configs := make(map[string]*rest.Config)
	var (
		f   os.FileInfo
		err error
	)

	if f, err = os.Stat(conf.VirtualGarden.TokenPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("token file not found: %w", err)
	}
	//check the length of the token file
	if f.Size() == 0 {
		return nil, fmt.Errorf("token file is empty")
	}

	// Create a rest.Config for the Virtual Garden
	restConfig := &rest.Config{
		Host:            fmt.Sprintf("https://api.%s.gardener.cloud.sap", conf.VirtualGarden.Environment),
		BearerTokenFile: conf.VirtualGarden.TokenPath,
	}
	configs[gardenerclient.VIRTUAL_GARDEN] = restConfig
	return configs, nil
}

func fetchContextName(name string, prefix string) string {
	if strings.HasPrefix(name, prefix+"-") {
		return strings.TrimPrefix(name, prefix+"-")
	}
	return name
}

func virtualGardenKubeconfig(conf *config.Config) string {
	if conf.VirtualGarden.Kubeconfig != "" {
		return conf.VirtualGarden.Kubeconfig
	}
	return os.Getenv("KUBECONFIG")
}
