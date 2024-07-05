// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
//
// Package kubesatoken implements utilities for retrieving Kubernetes service
// account tokens.
//
// The same utilities are used to construct a Web Identity Credentials Provider
// for AWS using the AWS STS, where a Kubernetes service account token is used
// to request temporary security credentials to the AWS services.
//
// In order to exchange the Kubernetes service account token for temporary
// security credentials it is expected that you already have an OpenID Connect
// IdP created in AWS and your Kubernetes cluster OpenID Metadata Provider
// endpoint is publically available.
//
// For more information, please refer to the following documentation.
//
// https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html
// https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html

package kubesatoken

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

const (
	// ProviderName specifies the name of the Credentials Provider.
	ProviderName = "kube_sa_token"
)

// ErrNoServiceAccount is an error, which is returned when the
// [TokenRetriever] was configured without a service account.
var ErrNoServiceAccount = errors.New("no service account specified")

// ErrNoNamespace is an error, which is returned when the [TokenRetriever] was
// configured without namespace for the service account.
var ErrNoNamespace = errors.New("no namespace specified")

// ErrNoSTSClient is an error, which is returned when creating a new credentials
// provider without the required AWS STS client.
var ErrNoSTSClient = errors.New("no STS client specified")

// ErrNoRoleARN is an error, which is returned when creating a new credentials
// provider without specifying a IAM Role ARN.
var ErrNoRoleARN = errors.New("no IAM Role ARN specified")

// ErrNoTokenRetriever is an error, which is returned when creating a new
// credentials provider, without specifying a [stscreds.IdentityTokenRetriever]
// implementation.
var ErrNoTokenRetriever = errors.New("no token retriever specified")

// TokenRetriever retrieves a service account token from Kubernetes, which can
// later be used to request temporary security credentials to access the AWS
// services using a Web Identity Credentials provider.
//
// The TokenRetriever implements the [stscreds.IdentityTokenRetriever]
// interface.
type TokenRetriever struct {
	// kubeClient is the Kubernetes client used by the token retriever.
	kubeClient *kubernetes.Clientset

	// namespace is the namespace of the service account
	namespace string

	// serviceAccount is the name of the service account for which the token
	// will be issued.
	serviceAccount string

	// kubeconfigFile is the path to the kubeconfig file to use, if empty it will
	// attempt to create a Kubernetes Client using in-cluster configuration.
	kubeconfigFile string

	// audiences specifies the audiences this token will be issued for.
	audiences []string

	// duration specifies the duration for which this token will be valid
	// for.
	duration time.Duration
}

var _ stscreds.IdentityTokenRetriever = &TokenRetriever{}

// GetToken creates and returns a new Kubernetes token.
func (t *TokenRetriever) GetToken(ctx context.Context) (*authenticationv1.TokenRequest, error) {
	expirationSeconds := t.duration.Seconds()
	req := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         t.audiences,
			ExpirationSeconds: ptr.To(int64(expirationSeconds)),
		},
	}
	out, err := t.kubeClient.CoreV1().ServiceAccounts(t.namespace).CreateToken(
		ctx,
		t.serviceAccount,
		req,
		metav1.CreateOptions{},
	)

	return out, err
}

// GetIdentityToken implements the [stscreds.IdentityTokenRetriever] interface.
func (t *TokenRetriever) GetIdentityToken() ([]byte, error) {
	out, err := t.GetToken(context.Background())
	if err != nil {
		return nil, err
	}

	return []byte(out.Status.Token), nil
}

// TokenRetrieverOption is a function which configures a [TokenRetriever]
// instance.
type TokenRetrieverOption func(*TokenRetriever)

// NewTokenRetriever creates a new [TokenRetriever] and configures it using the
// provided options.
func NewTokenRetriever(opts ...TokenRetrieverOption) (*TokenRetriever, error) {
	tokenRetriever := &TokenRetriever{}
	for _, opt := range opts {
		opt(tokenRetriever)
	}

	if tokenRetriever.serviceAccount == "" {
		return nil, ErrNoServiceAccount
	}

	if tokenRetriever.namespace == "" {
		return nil, ErrNoNamespace
	}

	// We have a configured client, we are done here.
	if tokenRetriever.kubeClient != nil {
		return tokenRetriever, nil
	}

	// ... otherwise, create a new Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", tokenRetriever.kubeconfigFile)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	tokenRetriever.kubeClient = client

	return tokenRetriever, nil
}

// WithNamespace configures the [TokenRetriever] to use the specified namespace.
func WithNamespace(namespace string) TokenRetrieverOption {
	opt := func(t *TokenRetriever) {
		t.namespace = namespace
	}

	return opt
}

// WithServiceAccount configures the [TokenRetriever] to use the specified
// service account name.
func WithServiceAccount(serviceAccount string) TokenRetrieverOption {
	opt := func(t *TokenRetriever) {
		t.serviceAccount = serviceAccount
	}

	return opt
}

// WithKubeconfig configures the [TokenRetriever] to use the given kubeconfig
// file.
func WithKubeconfig(kubeconfigFile string) TokenRetrieverOption {
	opt := func(t *TokenRetriever) {
		t.kubeconfigFile = kubeconfigFile
	}

	return opt
}

// WithAudiences configures the [TokenRetriever] to use the given audiences.
func WithAudiences(audiences []string) TokenRetrieverOption {
	opt := func(t *TokenRetriever) {
		t.audiences = audiences
	}

	return opt
}

// WithTokenExpiration configures the [TokenRetriever] to use the given token
// expiration duration.
func WithTokenExpiration(duration time.Duration) TokenRetrieverOption {
	opt := func(t *TokenRetriever) {
		t.duration = duration
	}

	return opt
}

// WithClient configures the [TokenRetriever] to use the given Kubernetes
// client.
func WithClient(client *kubernetes.Clientset) TokenRetrieverOption {
	opt := func(t *TokenRetriever) {
		t.kubeClient = client
	}

	return opt
}

// CredentialsProviderSpec provides the configuration settings for the Web
// Identity Credentials Provider.
type CredentialsProviderSpec struct {
	// Client is the API client used to make API calls to the AWS STS.
	Client *sts.Client

	// RoleARN is the IAM Role ARN to assume.
	RoleARN string

	// RoleSessionName is the name of the session, which uniquely identifies it
	RoleSessionName string

	// Duration specifies the expiry duration of the STS credentials.
	Duration time.Duration

	// TokenRetriever is the identity token retriever implementation to use.
	TokenRetriever stscreds.IdentityTokenRetriever
}

// NewCredentialsProvider creates a new [aws.CredentialsProvider] based on
// the provided spec.
func NewCredentialsProvider(spec *CredentialsProviderSpec) (aws.CredentialsProvider, error) {
	if spec.Client == nil {
		return nil, ErrNoSTSClient
	}

	if spec.RoleARN == "" {
		return nil, ErrNoRoleARN
	}

	if spec.TokenRetriever == nil {
		return nil, ErrNoTokenRetriever
	}

	opts := []func(o *stscreds.WebIdentityRoleOptions){
		func(o *stscreds.WebIdentityRoleOptions) {
			o.Duration = spec.Duration
		},
		func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = spec.RoleSessionName
		},
	}

	provider := stscreds.NewWebIdentityRoleProvider(
		spec.Client,
		spec.RoleARN,
		spec.TokenRetriever,
		opts...,
	)

	return provider, nil
}
