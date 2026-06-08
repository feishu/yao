package sse

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/session"
)

func ResolveIdentity(sid string) (Identity, error) {
	sid = strings.TrimSpace(sid)
	identity := Identity{SessionID: sid}
	if sid == "" {
		return identity, ErrIdentityNotFound
	}

	ss := session.Global().ID(sid)
	userID, err := ss.Get("user_id")
	if err != nil {
		return identity, err
	}
	if id := trimID(userID); id != "" {
		identity.UserID = id
		return identity, nil
	}

	uid, err := ss.Get("uid")
	if err != nil {
		return identity, err
	}
	if id := trimID(uid); id != "" {
		identity.UserID = id
		return identity, nil
	}

	return identity, ErrIdentityNotFound
}

func trimID(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
