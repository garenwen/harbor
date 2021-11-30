package baiduccr

import (
	"fmt"
	"net/http"

	"github.com/baidubce/bce-sdk-go/auth"
	"github.com/baidubce/bce-sdk-go/bce"
)

type Client struct {
	bce.Client
}

func newClient(ak, sk, sessionToken, region string) *Client {
	cred := &auth.BceCredentials{
		AccessKeyId:     ak,
		SecretAccessKey: sk,
		SessionToken:    sessionToken,
	}

	defaultSignOptions := &auth.SignOptions{
		HeadersToSign: auth.DEFAULT_HEADERS_TO_SIGN,
		ExpireSeconds: auth.DEFAULT_EXPIRE_SECONDS}

	defaultConf := &bce.BceClientConfiguration{
		Endpoint:                  getEndpoint(region),
		Region:                    region,
		UserAgent:                 "ccr-replication",
		Credentials:               cred,
		SignOption:                defaultSignOptions,
		Retry:                     bce.DEFAULT_RETRY_POLICY,
		ConnectionTimeoutInMillis: bce.DEFAULT_CONNECTION_TIMEOUT_IN_MILLIS,
		RedirectDisabled:          false}
	v1Signer := &auth.BceV1Signer{}

	return &Client{bce.NewBceClient(defaultConf, v1Signer)}
}

func (c *Client) CreateTemporaryToken(clusterID string, duration int) (string, error) {
	var resp TemporaryPasswordResponse
	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodPost).
		WithURL("/v1/instances/"+clusterID+"/credential").
		WithHeader("Content-Type", "application/json").
		WithBody(&TemporaryPasswordArgs{
			Duration: duration,
		}).WithResult(&resp).
		Do()

	return resp.Password, err
}

func (c *Client) GetNamespace(clusterID, namespace string) (*ProjectResult, error) {
	var resp ProjectResult
	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/" + clusterID + "/projects/" + namespace).
		WithResult(&resp).
		Do()

	return &resp, err
}

func (c *Client) CreateNamespace(clusterID string, arg *CreateProjectRequest) error {
	return bce.NewRequestBuilder(c).
		WithMethod(http.MethodPost).
		WithURL("/v1/instances/"+clusterID+"/projects").
		WithHeader("Content-Type", "application/json").
		WithBody(arg).
		Do()
}

func (c *Client) ListNamespaces(clusterID string, pageNo, pageSize int) (*ListProjectResponse, error) {
	var resp ListProjectResponse

	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/"+clusterID+"/projects").
		WithQueryParam("pageNo", fmt.Sprintf("%d", pageNo)).
		WithQueryParam("pageSize", fmt.Sprintf("%d", pageSize)).
		WithResult(&resp).
		Do()

	return &resp, err
}

func (c *Client) ListReposByNamespace(clusterID, namespace string, pageNo, pageSize int) (*ListRepositoryResponse, error) {
	var resp ListRepositoryResponse

	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/"+clusterID+"/projects/"+namespace+"/repositories").
		WithQueryParam("pageNo", fmt.Sprintf("%d", pageNo)).
		WithQueryParam("pageSize", fmt.Sprintf("%d", pageSize)).
		WithResult(&resp).
		Do()

	return &resp, err
}

func (c *Client) GetImage(clusterID, namespace, repo, tag string, pageNo, pageSize int) (*ListTagResponse, error) {
	var resp ListTagResponse

	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/"+clusterID+"/projects/"+namespace+"/repositories/"+repo+"/tags").
		WithQueryParamFilter("tagName", tag).
		WithQueryParam("pageNo", fmt.Sprintf("%d", pageNo)).
		WithQueryParam("pageSize", fmt.Sprintf("%d", pageSize)).
		WithResult(&resp).
		Do()
	return &resp, err
}

func (c *Client) DeleteImage(clusterID, namespace, repo, reference string) error {
	return bce.NewRequestBuilder(c).
		WithMethod(http.MethodDelete).
		WithURL("/v1/instances/" + clusterID + "/projects/" + namespace + "/repositories/" + repo + "/tags/" + reference).
		Do()
}
