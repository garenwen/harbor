package baiduccr

import (
	"net/http"
	"time"

	"github.com/goharbor/harbor/src/common/http/modifier"
	"github.com/goharbor/harbor/src/lib/log"
)

// Credential ...
type Credential modifier.Modifier

var _ Credential = &baiduAuthCredential{}

func (q *baiduAuthCredential) Modify(r *http.Request) (err error) {
	log.Debugf("[baiduCCR.Modify before]Host: %v", r.Host)
	if !q.isCacheTokenValid() {
		err = q.getTempInstanceToken()
		log.Debugf("baiduCCR.Modify.isCacheTokenValid.updateToken=%s, err=%v", q.cacheTokenExpiredAt, err)
		if err != nil {
			return
		}
	}
	r.SetBasicAuth(q.cacheTokener.username, q.cacheTokener.token)
	log.Debugf("[baiduCCR.Modify]Host: %v, header: %#v", r.Host, r.Header)
	return
}

func (q *baiduAuthCredential) isCacheTokenValid() (ok bool) {
	log.Debugf("[]baiduCCR.isCacheTokenValid: username: %s, expiredAt: %v", q.username, q.cacheTokenExpiredAt)
	if &q.cacheTokenExpiredAt == nil {
		return
	}
	if q.cacheTokener == nil {
		return
	}
	// refresh token in advanced
	if time.Now().After(q.cacheTokenExpiredAt.Add(-1 * time.Minute)) {
		return
	}
	return true
}

// Implements interface Credential
type baiduAuthCredential struct {
	registryID          string
	client              *cacheClient
	username            string
	cacheTokener        *temporaryTokener
	cacheTokenExpiredAt time.Time
}

type temporaryTokener struct {
	username string
	token    string
}

// NewAuth ...
func NewAuth(username, password string, registryID string, client *cacheClient) Credential {
	return &baiduAuthCredential{
		registryID: registryID,
		client:     client,
		username:   username,
		cacheTokener: &temporaryTokener{
			username: username,
			token:    password,
		},
		cacheTokenExpiredAt: time.Now().Add(time.Second),
	}
}

func (q *baiduAuthCredential) getTempInstanceToken() (err error) {
	var cli *Client
	cli, err = q.client.Get()
	if err != nil {
		return err
	}

	var passwd string
	passwd, err = cli.CreateTemporyToken(q.registryID, 2)
	if err != nil {
		return err
	}

	q.cacheTokenExpiredAt = time.Now().Add(2 * time.Hour)
	q.cacheTokener = &temporaryTokener{
		username: q.username,
		token:    passwd,
	}

	return
}
