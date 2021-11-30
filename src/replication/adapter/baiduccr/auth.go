package baiduccr

import (
	"github.com/goharbor/harbor/src/common/http/modifier"
	"net/http"
	"time"

	//"github.com/goharbor/harbor/src/common/http/modifier"
	"github.com/goharbor/harbor/src/lib/log"
)

// Credential ...
type Credential modifier.Modifier

var _ Credential = &baiduAuthCredential{}

// Implements interface Credential
type baiduAuthCredential struct {
	clusterID   string
	clientCache *clientCache

	username          string
	password          string
	passwordExpiredAt time.Time
}

// NewAuth ...
func NewAuth(username, password string, clusterID string, clientCache *clientCache) *baiduAuthCredential {
	return &baiduAuthCredential{
		clusterID:         clusterID,
		clientCache:       clientCache,
		username:          username,
		password:          password,
		passwordExpiredAt: time.Now().Add(time.Second),
	}
}

func (q *baiduAuthCredential) Modify(r *http.Request) error {
	log.Infof("[baidu-ccr.Modify before]Host: %v, header: %#v", r.Host, r.Header)
	if !q.isCacheTokenValid() {
		if err := q.getTempInstanceToken(); err != nil {
			log.Errorf("baidu-ccr.Modify.isCacheTokenValid.updateToken=%s, err=%v", q.passwordExpiredAt, err)
			return err
		}
	}
	r.SetBasicAuth(q.username, q.password)
	log.Infof("[baidu-ccr.Modify after]Host: %v, header: %#v", r.Host, r.Header)
	return nil
}

func (q *baiduAuthCredential) isCacheTokenValid() bool {
	log.Infof("[]baidu-ccr.isCacheTokenValid: username: %s, expiredAt: %v", q.username, q.passwordExpiredAt)
	if &q.passwordExpiredAt == nil {
		return false
	}
	if q.password == "" {
		return false
	}
	// refresh token in advanced
	if time.Now().After(q.passwordExpiredAt.Add(-1 * time.Minute)) {
		return false
	}
	return true
}

func (q *baiduAuthCredential) getTempInstanceToken() error {
	cli, err := q.clientCache.GetClient()
	if err != nil {
		return err
	}

	password, err := cli.CreateTemporaryToken(q.clusterID, 2)
	log.Infof("password: %s ,err: %s", password, err)

	if err != nil {
		return err
	}

	q.password = password
	q.passwordExpiredAt = time.Now().Add(2 * time.Hour)

	return nil
}
