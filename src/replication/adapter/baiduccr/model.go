package baiduccr

import (
	"time"
)

type PageInfo struct {
	Total    int `json:"total"`
	PageNo   int `json:"pageNo"`
	PageSize int `json:"pageSize"`
}

type CreateProjectRequest struct {
	ProjectName string
	Public      string
}

type RepositoryResult struct {

	// The count of the tags inside the repository
	TagCount int64 `json:"tagCount"`

	// The creation time of the repository
	// Format: date-time
	CreationTime time.Time `json:"creationTime"`

	// The description of the repository
	Description string `json:"description"`

	// The name of the repository
	RepositoryName string `json:"repositoryName"`

	// The count that the artifact inside the repository pulled
	PullCount int64 `json:"pullCount"`

	// The update time of the repository
	// Format: date-time
	UpdateTime time.Time `json:"updateTime"`

	// The path of the repository
	RepositoryPath string `json:"repositoryPath"`

	// The project name the repository
	ProjectName string `json:"projectName"`
}

type ProjectResult struct {

	// The total number of charts under this project.
	ChartCount int64 `json:"chartCount"`

	// The creation time of the project.
	// Format: date-time
	CreationTime time.Time `json:"creationTime"`

	// The name of the project.
	ProjectName string `json:"projectName"`

	// Project ID
	ProjectID int32 `json:"projectId"`

	// The number of the repositories under this project.
	RepoCount int64 `json:"repoCount"`

	// The update time of the project.
	// Format: date-time
	UpdateTime time.Time `json:"updateTime"`

	// Whether scan images automatically when pushing. The valid values are "true", "false".
	AutoScan *string `json:"autoScan"`

	// The public status of the project. The valid values are "true", "false".
	Public string `json:"public"`
}

type ListProjectResponse struct {
	PageInfo `json:",inline"`
	Items    []*ProjectResult `json:"items"`
}

type ListRepositoryResponse struct {
	PageInfo `json:",inline"`
	Items    []*RepositoryResult `json:"items"`
}

type ListTagResponse struct {
	PageInfo `json:",inline"`
	Items    []*TagResult `json:"items"`
}

type TagResult struct {
	// The name of the tag
	TagName string `json:"tagName"`
	// The digest of the artifact
	Digest string `json:"digest"`
	// The ID of the project that the artifact belongs to
	ProjectId int64 `json:"projectId"`
	// The latest pull time of the tag
	// Format: date-time
	PullTime time.Time `json:"pullTime"`
	// The push time of the tag
	// Format: date-time
	PushTime time.Time `json:"pushTime"`
	// The ID of the repository that the artifact belongs to
	RepositoryId int64 `json:"repositoryId"`
	// Architecture The architecture of repository
	Architecture string `json:"architecture"`
	// OS
	Os string `json:"os"`
	// Author
	Author string `json:"author"`
	// The type of the artifact, e.g. image, chart, etc
	Type string `json:"type"`
	// The size of the artifact
	Size int64 `json:"size"`
	// 漏洞扫描信息
	ScanOverview *ScanOverview `json:"scanOverview"`
}

type ScanOverview struct {
	// The status of the report generating process
	// Example: Success
	ScanStatus string `json:"scanStatus"`
	// The start time of the scan process that generating report
	// Example: 2006-01-02T14:04:05
	// Format: date-time
	StartTime string `json:"startTime"`
	// The end time of the scan process that generating report
	// Example: 2006-01-02T15:04:05
	// Format: date-time
	EndTime string `json:"endTime"`
	// id of the native scan report
	// Example: 5f62c830-f996-11e9-957f-0242c0a89008
	ReportId string `json:"reportId"`

	// 漏洞等级 Critical 危及 High 严重  Medium 中等 Low 较低
	Severity string `json:"severity"`
	// The number of the fixable vulnerabilities
	// Example: 100
	Fixable int64 `json:"fixable,omitempty"`
	// Numbers of the vulnerabilities with different severity
	// Example: {"Critical":5,"High":5}
	Summary map[string]int64 `json:"summary,omitempty"`
	// The total number of the found vulnerabilities
	// Example: 500
	Total int64 `json:"total,omitempty"`
}

type TemporaryPasswordResponse struct {
	Password string `json:"password,omitempty"`
}

type TemporaryPasswordArgs struct {
	Duration int `json:"duration,omitempty" binding:"required,min=1,max=24"`
}
