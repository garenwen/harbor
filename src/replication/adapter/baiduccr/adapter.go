package baiduccr

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/docker/distribution/registry/client/auth/challenge"
	commonhttp "github.com/goharbor/harbor/src/common/http"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/registry/auth/bearer"
	adp "github.com/goharbor/harbor/src/replication/adapter"
	"github.com/goharbor/harbor/src/replication/adapter/native"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/goharbor/harbor/src/replication/util"
)

var (
	errInvalidCCREndpoint = errors.New("[baidu-ccr.newAdapter] Invalid baidu CCR instance endpoint")
	//errPingCCREndpointFailed = errors.New("[baidu-ccr.newAdapter] Ping baidu CCR instance endpoint failed")
	errInvalidCCRCredential = errors.New("[baidu-ccr.newAdapter] Invalid baidu CCR credential")
)

func init() {
	if err := adp.RegisterFactory(model.RegistryTypeBaiduCcr, new(factory)); err != nil {
		log.Errorf("failed to register factory for %s: %v", model.RegistryTypeBaiduCcr, err)
		return
	}
	log.Infof("the factory for adapter %s registered", model.RegistryTypeBaiduCcr)
}

type factory struct{}

/**
	* Implement Factory Interface
**/
var _ adp.Factory = &factory{}

// Create ...
func (f *factory) Create(r *model.Registry) (adp.Adapter, error) {
	return newAdapter(r)
}

// AdapterPattern ...
func (f *factory) AdapterPattern() *model.AdapterPattern {
	return getAdapterInfo()
}

func getAdapterInfo() *model.AdapterPattern {
	return nil
}

type adapter struct {
	*native.Adapter
	instanceID *string
	regionName *string
	client     *commonhttp.Client
	registry   *model.Registry

	clientCache *clientCache
}

/**
	* Implement Adapter Interface
**/
var _ adp.Adapter = &adapter{}

func newAdapter(registry *model.Registry) (a *adapter, err error) {
	// Query baidu CCR instance info via endpoint.
	var registryURL *url.URL
	registryURL, _ = url.Parse(registry.URL)

	if !strings.HasSuffix(registryURL.Host, "baidubce.com") {
		log.Errorf("[baidu-ccr.newAdapter] errInvalidCCREndpoint=%v", err)
		return nil, errInvalidCCREndpoint
	}

	if registry.Credential == nil || registry.Credential.Type != model.CredentialTypeBasic {
		err = errInvalidCCRCredential
		log.Errorf("[baidu-ccr.newAdapter] credential is in wrong type")
		return
	}

	realm, service, err := ping(registry)
	if err != nil {
		log.Errorf("[baidu-ccr.newAdapter] ping failed. error=%v", err)
		return
	}

	region := getRegionFromHost(registryURL.Host)
	instanceID := getInstanceIDFromHost(registryURL.Host)
	clientCache := newClientCache(instanceID, registry.Credential.AccessKey, region)

	var credential = NewAuth(registry.Credential.AccessKey, registry.Credential.AccessSecret, instanceID, clientCache)
	var transport = util.GetHTTPTransport(registry.Insecure)
	var authorizer = bearer.NewAuthorizer(realm, service, credential, transport)

	return &adapter{
		registry:    registry,
		instanceID:  &instanceID,
		regionName:  &region,
		clientCache: clientCache,
		client: commonhttp.NewClient(
			&http.Client{
				Transport: transport,
			},
			credential,
		),
		Adapter: native.NewAdapterWithAuthorizer(registry, authorizer),
	}, nil
}

func ping(registry *model.Registry) (string, string, error) {
	client := &http.Client{
		Transport: util.GetHTTPTransport(registry.Insecure),
	}

	resp, err := client.Get(registry.URL + "/v2/")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	challenges := challenge.ResponseChallenges(resp)
	for _, challenge := range challenges {
		if challenge.Scheme == "bearer" {
			return challenge.Parameters["realm"], challenge.Parameters["service"], nil
		}
	}
	return "", "", fmt.Errorf("[baidu-ccr.ping] bearer auth scheme isn't supported: %v", challenges)
}

func (a *adapter) Info() (info *model.RegistryInfo, err error) {
	info = &model.RegistryInfo{
		Type: model.RegistryTypeBaiduCcr,
		SupportedResourceTypes: []model.ResourceType{
			model.ResourceTypeImage,
			model.ResourceTypeChart,
		},
		SupportedResourceFilters: []*model.FilterStyle{
			{
				Type:  model.FilterTypeName,
				Style: model.FilterStyleTypeText,
			},
			{
				Type:  model.FilterTypeTag,
				Style: model.FilterStyleTypeText,
			},
		},
		SupportedTriggers: []model.TriggerType{
			model.TriggerTypeManual,
			model.TriggerTypeScheduled,
		},
	}
	return
}

func (a *adapter) PrepareForPush(resources []*model.Resource) (err error) {
	for _, resource := range resources {
		if resource == nil {
			return errors.New("the resource cannot be null")
		}
		if resource.Metadata == nil {
			return errors.New("[baidu-ccr.PrepareForPush] the metadata of resource cannot be null")
		}
		if resource.Metadata.Repository == nil {
			return errors.New("[baidu-ccr.PrepareForPush] the namespace of resource cannot be null")
		}
		if len(resource.Metadata.Repository.Name) == 0 {
			return errors.New("[baidu-ccr.PrepareForPush] the name of the namespace cannot be null")
		}
		var paths = strings.Split(resource.Metadata.Repository.Name, "/")
		var namespace = paths[0]

		err = a.createPrivateNamespace(namespace)
		if err != nil {
			return
		}
		return
	}

	return
}

func (a *adapter) createPrivateNamespace(namespace string) error {
	cli, err := a.clientCache.GetClient()
	if err != nil {
		return errors.New(fmt.Sprintf("[baidu-ccr.createPrivateNamespace] get client failed: %s", err))
	}

	isExist, err := a.isNamespaceExists(namespace)
	if err != nil {
		return err
	}

	if !isExist {
		return cli.CreateNamespace(*a.instanceID, &CreateProjectRequest{
			ProjectName: namespace,
			Public:      "false",
		})
	}

	return nil
}

func (a *adapter) isNamespaceExists(namespace string) (bool, error) {
	cli, err := a.clientCache.GetClient()
	if err != nil {
		return false, errors.New(fmt.Sprintf("[baidu-ccr.isNamespaceExists] get client failed: %s", err))
	}

	_, err = cli.GetNamespace(*a.instanceID, namespace)
	var bceErr *bce.BceServiceError
	if err != nil && errors.As(err, &bceErr) {
		if bceErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
	}

	if err != nil {
		return false, fmt.Errorf("[baidu-ccr.isNamespaceExists] get namespace failed: %w", err)
	}

	return true, nil
}

func (a *adapter) listAllNamespaces() ([]string, error) {
	cli, err := a.clientCache.GetClient()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[baidu-ccr.listAllNamespaces] get client failed: %s", err))
	}

	pageNo, pageSize := 1, 100
	result := make([]string, 0)
	for {
		resp, err := cli.ListNamespaces(*a.instanceID, pageNo, pageSize)
		if err != nil {
			return nil, fmt.Errorf("[baidu-ccr.listAllNamespaces] list namespaces: %w", err)
		}

		if resp.Items == nil {
			return result, nil
		}

		for _, v := range resp.Items {
			result = append(result, v.ProjectName)
		}

		if len(resp.Items) == 0 || (pageNo*pageSize+len(resp.Items)) >= resp.Total {
			break
		}

		pageNo++
	}

	return result, nil
}

func (a *adapter) listReposByNamespace(namespace string) ([]*RepositoryResult, error) {
	cli, err := a.clientCache.GetClient()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[baidu-ccr.ListReposByNamespace] get client failed: %s", err))
	}

	pageNo, pageSize := 1, 100
	result := make([]*RepositoryResult, 0)
	for {
		resp, err := cli.ListReposByNamespace(*a.instanceID, namespace, pageNo, pageSize)
		if err != nil {
			return nil, fmt.Errorf("[baidu-ccr.ListReposByNamespace] list namespaces: %w", err)
		}

		if len(resp.Items) != 0 {
			result = append(result, resp.Items...)
		}

		if len(resp.Items) == 0 || (pageNo*pageSize+len(resp.Items)) >= resp.Total {
			break
		}

		pageNo++
	}

	return result, nil
}

func (a *adapter) getImages(namespace, repo, tag string) ([]*TagResult, []string, error) {
	cli, err := a.clientCache.GetClient()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("[baidu-ccr.getImages] get client failed: %s", err))
	}

	pageNo, pageSize := 1, 100
	result := make([]*TagResult, 0)
	images := make([]string, 0)
	for {
		resp, err := cli.GetImage(*a.instanceID, namespace, repo, tag, pageNo, pageSize)
		if err != nil {
			return nil, nil, fmt.Errorf("[baidu-ccr.getImages] get image: %w", err)
		}

		if resp.Items == nil {
			return result, images, nil
		}

		for _, v := range resp.Items {
			result = append(result, v)
			images = append(images, v.TagName)
		}

		if len(resp.Items) == 0 || (pageNo*pageSize+len(resp.Items)) >= resp.Total {
			break
		}

		pageNo++
	}

	return result, images, nil
}

func (a *adapter) deleteImage(namespace, repo, tag string) error {
	cli, err := a.clientCache.GetClient()
	if err != nil {
		return errors.New(fmt.Sprintf("[baidu-ccr.deleteImage] get client failed: %s", err))
	}

	return cli.DeleteImage(*a.instanceID, namespace, repo, tag)
}
