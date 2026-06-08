package sse

import (
	"errors"
	"testing"

	"github.com/yaoapp/gou/session"
)

func TestResolveIdentityUsesUserIDFirst(t *testing.T) {
	sid := session.ID()
	err := session.Global().ID(sid).Set("user_id", 831)
	if err != nil {
		t.Fatalf("set user_id: %v", err)
	}
	err = session.Global().ID(sid).Set("uid", "fallback")
	if err != nil {
		t.Fatalf("set uid: %v", err)
	}

	identity, err := ResolveIdentity(sid)
	if err != nil {
		t.Fatalf("ResolveIdentity returned error: %v", err)
	}

	if identity.SessionID != sid {
		t.Fatalf("SessionID = %q, want %q", identity.SessionID, sid)
	}
	if identity.UserID != "831" {
		t.Fatalf("UserID = %q, want %q", identity.UserID, "831")
	}
}

func TestResolveIdentityFallsBackToUID(t *testing.T) {
	sid := session.ID()
	err := session.Global().ID(sid).Set("uid", "u-100")
	if err != nil {
		t.Fatalf("set uid: %v", err)
	}

	identity, err := ResolveIdentity(sid)
	if err != nil {
		t.Fatalf("ResolveIdentity returned error: %v", err)
	}

	if identity.UserID != "u-100" {
		t.Fatalf("UserID = %q, want %q", identity.UserID, "u-100")
	}
}

func TestResolveIdentityReturnsErrIdentityNotFound(t *testing.T) {
	sid := session.ID()

	identity, err := ResolveIdentity(sid)
	if !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrIdentityNotFound)
	}
	if identity.UserID != "" {
		t.Fatalf("UserID = %q, want empty", identity.UserID)
	}
}
