// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
//
// Package kubesatoken implements utilities for retrieving Kubernetes service
// account tokens.
//
// The token retriever is meant to be plugged-in to an AWS Web Identity
// Credentials Provider, so that short-lived JWT tokens can be exchanged for
// temporary security credentials when accessing AWS resources.

package kubesatoken

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

const (
	// TokenRetrieverName specifies the name of the Token Retriever.
	TokenRetrieverName = "kube_sa_token"
)

// ErrNoServiceAccount is an error, which is returned when the
// [TokenRetriever] was configured without a service account.
var ErrNoServiceAccount = errors.New("no service account specified")

// ErrNoNamespace is an error, which is returned when the [TokenRetriever] was
// configured without namespace for the service account.
var ErrNoNamespace = errors.New("no namespace specified")

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
