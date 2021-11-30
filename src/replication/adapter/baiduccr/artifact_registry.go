package baiduccr

import (
	"fmt"
	"strings"

	"github.com/goharbor/harbor/src/common/utils"
	"github.com/goharbor/harbor/src/lib/log"
	adp "github.com/goharbor/harbor/src/replication/adapter"
	"github.com/goharbor/harbor/src/replication/filter"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/goharbor/harbor/src/replication/util"
)

var (
	concurrencyNumber = 10
)

/**
	* Implement ArtifactRegistry Interface
**/
var _ adp.ArtifactRegistry = &adapter{}

func filterToPatterns(filters []*model.Filter) (namespacePattern, repoPattern, tagsPattern string) {
	for _, f := range filters {
		if f.Type == model.FilterTypeName {
			repoPattern = f.Value.(string)
		}
		if f.Type == model.FilterTypeTag {
			tagsPattern = f.Value.(string)
		}
	}
	namespacePattern = strings.Split(repoPattern, "/")[0]
	return
}

func (a *adapter) FetchArtifacts(filters []*model.Filter) (resources []*model.Resource, err error) {
	// get filter pattern
	var namespacePattern, repoPattern, tagsPattern = filterToPatterns(filters)
	log.Debugf("[baidu-ccr.FetchArtifacts] namespacePattern=%s repoPattern=%s tagsPattern=%s", namespacePattern, repoPattern, tagsPattern)

	// 1. list namespaces
	var namespaces []string
	namespaces, err = a.listCandidateNamespaces(namespacePattern)
	if err != nil {
		return
	}
	log.Debugf("[baidu-ccr.FetchArtifacts] namespaces=%v", namespaces)

	// 2. list repos
	// var filteredRepos []tcr.TcrRepositoryInfo
	var repos []*model.Repository
	var repositories []*model.Repository
	for _, ns := range namespaces {
		repos, err := a.listReposByNamespace(ns)
		if err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			continue
		}
		for _, repo := range repos {
			repositories = append(repositories, &model.Repository{
				Name: repo.RepositoryName,
			})
		}
	}
	repos, _ = filter.DoFilterRepositories(repositories, filters)
	log.Debugf("[baidu-ccr.FetchArtifacts] filteredRepos=%d", len(repos))

	// 4. list images
	var rawResources = make([]*model.Resource, len(repos))
	runner := utils.NewLimitedConcurrentRunner(concurrencyNumber)

	for i, r := range repos {
		// !copy
		index := i
		repo := r

		runner.AddTask(func() error {
			var images []string
			repoArr := strings.Split(repo.Name, "/")
			_, images, err = a.getImages(repoArr[0], strings.Join(repoArr[1:], "/"), "")
			if err != nil {
				return fmt.Errorf("[baidu-ccr.FetchArtifacts.listImages] runner=%d repo=%s, error=%v", index, repo.Name, err)
			}

			var filteredImages []string
			if tagsPattern != "" {
				for _, image := range images {
					var ok bool
					ok, err = util.Match(tagsPattern, image)
					if err != nil {
						return fmt.Errorf("[baidu-ccr.FetchArtifacts.matchImage] image='%s', error=%v", image, err)
					}
					if ok {
						filteredImages = append(filteredImages, image)
					}
				}
			} else {
				filteredImages = images
			}

			log.Debugf("[baidu-ccr.FetchArtifacts] repo=%s, images=%v, filteredImages=%v", repo.Name, images, filteredImages)

			if len(filteredImages) > 0 {
				rawResources[index] = &model.Resource{
					Type:     model.ResourceTypeImage,
					Registry: a.registry,
					Metadata: &model.ResourceMetadata{
						Repository: &model.Repository{
							Name: repo.Name,
						},
						Vtags: filteredImages,
					},
				}
			}

			return nil
		})
	}
	if err = runner.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch artifacts: %v", err)
	}

	for _, res := range rawResources {
		if res != nil {
			resources = append(resources, res)
		}
	}
	log.Debugf("[baidu-ccr.FetchArtifacts] resources.size=%d", len(resources))

	return
}

func (a *adapter) listCandidateNamespaces(namespacePattern string) (namespaces []string, err error) {
	// filter namespaces
	if len(namespacePattern) > 0 {
		if nms, ok := util.IsSpecificPathComponent(namespacePattern); ok {
			// Check is exist
			var exist bool
			for _, ns := range nms {
				exist, err = a.isNamespaceExists(ns)
				if err != nil {
					return
				}
				if !exist {
					continue
				}
				namespaces = append(namespaces, nms...)
			}
		}
	}

	if len(namespaces) > 0 {
		log.Debugf("[baidu-ccr.listCandidateNamespaces] pattern=%s, namespaces=%v", namespacePattern, namespaces)
		return namespaces, nil
	}

	// list all
	return a.listAllNamespaces()
}

func (a *adapter) DeleteManifest(repository, reference string) (err error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("tcr only support repo in format <namespace>/<name>, but got: %s", repository)
	}
	log.Warningf("[baidu-ccr.DeleteManifest] namespace=%s, repository=%s, tag=%s", parts[0], parts[1], reference)

	return a.deleteImage(parts[0], parts[1], reference)
}
