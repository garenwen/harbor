package baiduccr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/goharbor/harbor/src/lib/log"
)

type clientCache struct {
	region     string
	instanceID string
	username   string
	expiredAt  time.Time

	client *Client
}

func newClientCache(instanceID, username, region string) *clientCache {
	return &clientCache{
		instanceID: instanceID,
		username:   username,
		region:     region,
		expiredAt:  time.Now(),
	}
}

func (c *clientCache) GetClient() (*Client, error) {

	if c.expiredAt.UTC().Add(-15 * time.Minute).After(time.Now().UTC()) {
		return c.client, nil
	}

	// Change the token 15 minutes in advance
	log.Debugf("client expired, start refresh it, username is :%s, expiredAt: %s", c.username, c.expiredAt)
	content, err := ioutil.ReadFile(credentialPath + "/" + c.instanceID)
	if err != nil {
		return nil, err
	}

	var usernamesMap map[string]*struct {
		AK           string    `json:"ak,omitempty"`
		SK           string    `json:"sk,omitempty"`
		SessionToken string    `json:"sessionToken,omitempty"`
		ExpiredAt    time.Time `json:"expiredAt,omitempty"`
	}
	err = json.Unmarshal(content, &usernamesMap)
	if err != nil {
		return nil, err
	}

	cred := usernamesMap[c.username]
	if cred == nil {
		return nil, fmt.Errorf("cannot found suitale credential for: %v", c.username)
	}

	c.expiredAt = cred.ExpiredAt
	c.client = newClient(cred.AK, cred.SK, cred.SessionToken, c.region)

	return c.client, nil
}
