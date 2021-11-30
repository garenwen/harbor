package baiduccr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/baidubce/bce-sdk-go/auth"
	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/goharbor/harbor/src/lib/log"
)

type CredentialSecret struct {
	AK           string    `json:"ak,omitempty"`
	SK           string    `json:"sk,omitempty"`
	SessionToken string    `json:"sessionToken,omitempty"`
	ExpiredAt    time.Time `json:"expiredAt,omitempty"`
}

type cacheClient struct {
	cli *Client

	clusterID string
	username  string
	region    string
	endpoint  string
	expiredAt time.Time
}

func newCacheClient(clusterID, userID, region string) *cacheClient {
	return &cacheClient{
		cli:       nil,
		clusterID: clusterID,
		username:  userID,
		region:    region,
		expiredAt: time.Now(),
	}
}

func (c *cacheClient) Get() (*Client, error) {
	// 提前15分钟换token
	if c.expiredAt.UTC().Add(-15 * time.Minute).Before(time.Now().UTC()) {
		log.Debugf("client expired, start refresh it, username is :%s, expiredAt: %s", c.username, c.expiredAt)
		content, err := ioutil.ReadFile(credentialPath + "/" + c.clusterID)
		if err != nil {
			return nil, err
		}

		var clsConf map[string]*CredentialSecret
		err = json.Unmarshal(content, &clsConf)
		if err != nil {
			return nil, err
		}

		cred := clsConf[c.username]
		if cred == nil {
			return nil, fmt.Errorf("cannot found suitale credential for: %v", c.username)
		}

		c.expiredAt = cred.ExpiredAt
		c.cli = NewClient(cred.AK, cred.SK, cred.SessionToken, c.region, getEndpoint(c.region))
	}

	return c.cli, nil
}

type Client struct {
	bce.Client
}

func NewClient(ak, sk, sessionToken, region, endpoint string) *Client {
	cred := &auth.BceCredentials{
		AccessKeyId:     ak,
		SecretAccessKey: sk,
		SessionToken:    sessionToken,
	}

	defaultSignOptions := &auth.SignOptions{
		HeadersToSign: auth.DEFAULT_HEADERS_TO_SIGN,
		ExpireSeconds: auth.DEFAULT_EXPIRE_SECONDS}

	defaultConf := &bce.BceClientConfiguration{
		Endpoint:                  endpoint,
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

func (c *Client) CreateTemporyToken(instanceID string, duration int) (string, error) {
	var resp TemporaryPasswordResponse
	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodPost).
		WithURL("/v1/instances/"+instanceID+"/credential").
		WithHeader("Content-Type", "application/json").
		WithBody(&TemporaryPasswordArgs{
			Duration: duration,
		}).WithResult(&resp).
		Do()

	return resp.Password, err
}

func (c *Client) GetNamespace(instanceID, namespace string) (*ProjectResult, error) {
	var resp ProjectResult
	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/" + instanceID + "/projects/" + namespace).
		WithResult(&resp).
		Do()

	return &resp, err
}

func (c *Client) CreateNamespace(instanceID string, arg *CreateProjectRequest) error {
	return bce.NewRequestBuilder(c).
		WithMethod(http.MethodPost).
		WithURL("/v1/instances/"+instanceID+"/projects").
		WithHeader("Content-Type", "application/json").
		WithBody(arg).
		Do()
}

func (c *Client) ListNamespaces(instanceID string, pageNo, pageSize int) (*ListProjectResponse, error) {
	var resp ListProjectResponse

	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/"+instanceID+"/projects").
		WithQueryParam("pageNo", fmt.Sprintf("%d", pageNo)).
		WithQueryParam("pageSize", fmt.Sprintf("%d", pageSize)).
		WithResult(&resp).
		Do()

	return &resp, err
}

func (c *Client) ListReposByNamespace(instanceID, namespace string, pageNo, pageSize int) (*ListRepositoryResponse, error) {
	var resp ListRepositoryResponse

	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/"+instanceID+"/projects/"+namespace+"/repositories").
		WithQueryParam("pageNo", fmt.Sprintf("%d", pageNo)).
		WithQueryParam("pageSize", fmt.Sprintf("%d", pageSize)).
		WithResult(&resp).
		Do()

	return &resp, err
}

func (c *Client) GetImage(instanceID, namespace, repo, tag string, pageNo, pageSize int) (*ListTagResponse, error) {
	var resp ListTagResponse

	err := bce.NewRequestBuilder(c).
		WithMethod(http.MethodGet).
		WithURL("/v1/instances/"+instanceID+"/projects/"+namespace+"/repositories/"+repo+"/tags").
		WithQueryParamFilter("tagName", tag).
		WithQueryParam("pageNo", fmt.Sprintf("%d", pageNo)).
		WithQueryParam("pageSize", fmt.Sprintf("%d", pageSize)).
		WithResult(&resp).
		Do()
	return &resp, err
}

func (c *Client) DeleteImage(instanceID, namespace, repo, reference string) error {
	return bce.NewRequestBuilder(c).
		WithMethod(http.MethodDelete).
		WithURL("/v1/instances/" + instanceID + "/projects/" + namespace + "/repositories/" + repo + "/tags/" + reference).
		Do()
}
