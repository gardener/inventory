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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
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
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/core/config"
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
	if conf.AWS.Region == "" && conf.AWS.DefaultRegion == "" {
		return errNoAWSRegion
	}

	if conf.AWS.Credentials.TokenRetriever == "" {
		return errNoAWSTokenRetriever
	}

	supportedTokenRetrievers := []string{
		config.DefaultAWSTokenRetriever,
		kubesatoken.TokenRetrieverName,
		tokenfile.TokenRetrieverName,
	}

	if !slices.Contains(supportedTokenRetrievers, conf.AWS.Credentials.TokenRetriever) {
		return fmt.Errorf("%w: %s", errUnknownAWSTokenRetriever, conf.AWS.Credentials.TokenRetriever)
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
// temporary security credentials when accessing AWS resources.
func newKubeSATokenCredentialsProvider(conf *config.Config) (aws.CredentialsProvider, error) {
	tokenRetriever, err := kubesatoken.NewTokenRetriever(
		kubesatoken.WithKubeconfig(conf.AWS.Credentials.KubeSATokenRetriever.Kubeconfig),
		kubesatoken.WithServiceAccount(conf.AWS.Credentials.KubeSATokenRetriever.ServiceAccount),
		kubesatoken.WithNamespace(conf.AWS.Credentials.KubeSATokenRetriever.Namespace),
		kubesatoken.WithAudiences(conf.AWS.Credentials.KubeSATokenRetriever.Audiences),
		kubesatoken.WithTokenExpiration(conf.AWS.Credentials.KubeSATokenRetriever.Duration),
	)

	if err != nil {
		return nil, err
	}

	providerSpec := &provider.Spec{
		Client:          newAWSSTSClient(conf),
		RoleARN:         conf.AWS.Credentials.KubeSATokenRetriever.RoleARN,
		RoleSessionName: conf.AWS.Credentials.KubeSATokenRetriever.RoleSessionName,
		Duration:        conf.AWS.Credentials.KubeSATokenRetriever.Duration,
		TokenRetriever:  tokenRetriever,
	}

	return provider.New(providerSpec)
}

// newTokenFileCredentialsProvider creates a new [aws.CredentialsProvider],
// which reads a JWT token from a specified path and exchanges the token for
// temporary security credentials when accessing AWS resources.
func newTokenFileCredentialsProvider(conf *config.Config) (aws.CredentialsProvider, error) {
	tokenRetriever, err := tokenfile.NewTokenRetriever(
		tokenfile.WithPath(conf.AWS.Credentials.TokenFileRetriever.Path),
	)

	if err != nil {
		return nil, err
	}

	providerSpec := &provider.Spec{
		Client:          newAWSSTSClient(conf),
		RoleARN:         conf.AWS.Credentials.TokenFileRetriever.RoleARN,
		RoleSessionName: conf.AWS.Credentials.TokenFileRetriever.RoleSessionName,
		Duration:        conf.AWS.Credentials.TokenFileRetriever.Duration,
		TokenRetriever:  tokenRetriever,
	}

	return provider.New(providerSpec)
}

// loadAWSDefaultConfig loads the AWS configurations and returns it.
func loadAWSDefaultConfig(ctx context.Context, conf *config.Config) (aws.Config, error) {
	opts := []func(o *awsconfig.LoadOptions) error{
		awsconfig.WithRegion(conf.AWS.Region),
		awsconfig.WithDefaultRegion(conf.AWS.DefaultRegion),
		awsconfig.WithAppID(conf.AWS.AppID),
	}

	switch conf.AWS.Credentials.TokenRetriever {
	case config.DefaultAWSTokenRetriever:
		// Load shared credentials config only
		break
	case kubesatoken.TokenRetrieverName:
		credsProvider, err := newKubeSATokenCredentialsProvider(conf)
		if err != nil {
			return aws.Config{}, err
		}
		opts = append(opts, awsconfig.WithCredentialsProvider(credsProvider))
	case tokenfile.TokenRetrieverName:
		credsProvider, err := newTokenFileCredentialsProvider(conf)
		if err != nil {
			return aws.Config{}, err
		}
		opts = append(opts, awsconfig.WithCredentialsProvider(credsProvider))
	default:
		return aws.Config{}, errUnknownAWSTokenRetriever
	}

	return awsconfig.LoadDefaultConfig(ctx, opts...)
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

	errHandler := func(ctx context.Context, task *asynq.Task, err error) {
		taskID, _ := asynq.GetTaskID(ctx)
		taskName := task.Type()
		queueName, _ := asynq.GetQueueName(ctx)
		retried, _ := asynq.GetRetryCount(ctx)
		slog.Error(
			"task failed",
			"id", taskID,
			"name", taskName,
			"queue", queueName,
			"retry", retried,
			"reason", err,
		)
	}
	config := asynq.Config{
		Concurrency:  conf.Worker.Concurrency,
		LogLevel:     logLevel,
		ErrorHandler: asynq.ErrorHandlerFunc(errHandler),
	}

	server := asynq.NewServer(redisClientOpt, config)

	return server
}

// newLoggingMiddleware returns a new [asynq.MiddlewareFunc] which logs each
// received task.
func newLoggingMiddleware() asynq.MiddlewareFunc {
	middleware := func(handler asynq.Handler) asynq.Handler {
		mw := func(ctx context.Context, task *asynq.Task) error {
			taskID, _ := asynq.GetTaskID(ctx)
			queueName, _ := asynq.GetQueueName(ctx)
			taskName := task.Type()
			slog.Info(
				"received task",
				"id", taskID,
				"queue", queueName,
				"name", taskName,
			)
			start := time.Now()
			err := handler.ProcessTask(ctx, task)
			elapsed := time.Since(start)
			slog.Info(
				"task finished",
				"id", taskID,
				"queue", queueName,
				"name", taskName,
				"duration", elapsed,
			)
			return err
		}

		return asynq.HandlerFunc(mw)
	}

	return asynq.MiddlewareFunc(middleware)
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

	configs := make(map[string]*rest.Config)

	// Attempt to read the kubeconfig from the configuration file
	kubeconfig := fetchKubeconfig(conf)

	// If the kubeconfig is not set, assume we are running in a Kubernetes cluster
	if kubeconfig == "" {
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
		//TODO: Most likely we are not going to deploy in the virtual-garden cluster
		// so we need to supply the virtual-garden cluster config via the configuration
		configs[clients.VIRTUAL_GARDEN] = inClusterConfig
		return configs, nil
	}

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
	return configs, nil
}

func fetchContextName(name string, prefix string) string {
	if strings.HasPrefix(name, prefix+"-") {
		return strings.TrimPrefix(name, prefix+"-")
	}
	return name
}

func fetchKubeconfig(conf *config.Config) string {
	if conf.VirtualGarden.Kubeconfig != "" {
		return conf.VirtualGarden.Kubeconfig
	}
	return os.Getenv("KUBECONFIG")
}
