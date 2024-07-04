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

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

// ErrNoServiceAccount is returned when the [KubeSATokenRetriever] was
// configured without a service account.
var ErrNoServiceAccount = errors.New("no service account specified")

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
type TokenRetrieverOption func(k *TokenRetriever)

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
