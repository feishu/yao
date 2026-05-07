package rtc

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sort"
	"time"
)

const (
	tokenVersion  = "001"
	appIDLength   = 24
	defaultMaxInt = 99999999
)

type privilege uint16

const (
	privPublishStream privilege = iota
	privPublishAudioStream
	privPublishVideoStream
	privPublishDataStream
	privSubscribeStream
)

type accessToken struct {
	appID      string
	appKey     string
	roomID     string
	userID     string
	issuedAt   uint32
	expireAt   uint32
	nonce      uint32
	privileges map[uint16]uint32
}

func newAccessToken(appID, appKey, roomID, userID string) (*accessToken, error) {
	nonce, err := randomNonce()
	if err != nil {
		return nil, err
	}

	return &accessToken{
		appID:      appID,
		appKey:     appKey,
		roomID:     roomID,
		userID:     userID,
		issuedAt:   uint32(time.Now().Unix()),
		nonce:      nonce,
		privileges: make(map[uint16]uint32),
	}, nil
}

func randomNonce() (uint32, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(defaultMaxInt))
	if err != nil {
		return 0, err
	}
	return uint32(n.Int64() + 1), nil
}

func (t *accessToken) expireTime(expireAt time.Time) {
	if !expireAt.IsZero() {
		t.expireAt = uint32(expireAt.Unix())
	}
}

func (t *accessToken) addPrivilege(p privilege, expireAt time.Time) {
	expire := uint32(expireAt.Unix())
	if expireAt.IsZero() {
		expire = 0
	}

	t.privileges[uint16(p)] = expire
	if p == privPublishStream {
		t.privileges[uint16(privPublishAudioStream)] = expire
		t.privileges[uint16(privPublishVideoStream)] = expire
		t.privileges[uint16(privPublishDataStream)] = expire
	}
}

func (t *accessToken) serialize() (string, error) {
	if len(t.appID) != appIDLength {
		return "", fmt.Errorf("invalid app id length: %d", len(t.appID))
	}

	msg, sign, err := t.pack()
	if err != nil {
		return "", err
	}

	bufContent := new(bytes.Buffer)
	if err := packString(bufContent, msg); err != nil {
		return "", err
	}
	if err := packString(bufContent, sign); err != nil {
		return "", err
	}

	return tokenVersion + t.appID + base64.StdEncoding.EncodeToString(bufContent.Bytes()), nil
}

func (t *accessToken) pack() (string, string, error) {
	buf := new(bytes.Buffer)
	if err := packUint32(buf, t.nonce); err != nil {
		return "", "", err
	}
	if err := packUint32(buf, t.issuedAt); err != nil {
		return "", "", err
	}
	if err := packUint32(buf, t.expireAt); err != nil {
		return "", "", err
	}
	if err := packString(buf, t.roomID); err != nil {
		return "", "", err
	}
	if err := packString(buf, t.userID); err != nil {
		return "", "", err
	}
	if err := packMapUint32(buf, t.privileges); err != nil {
		return "", "", err
	}

	msg := buf.Bytes()
	mac := hmac.New(sha256.New, []byte(t.appKey))
	mac.Write(msg)

	return string(msg), string(mac.Sum(nil)), nil
}

func packUint16(w io.Writer, n uint16) error {
	return binary.Write(w, binary.LittleEndian, n)
}

func packUint32(w io.Writer, n uint32) error {
	return binary.Write(w, binary.LittleEndian, n)
}

func packString(w io.Writer, s string) error {
	if err := packUint16(w, uint16(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func packMapUint32(w io.Writer, values map[uint16]uint32) error {
	if err := packUint16(w, uint16(len(values))); err != nil {
		return err
	}

	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)

	for _, key := range keys {
		if err := packUint16(w, uint16(key)); err != nil {
			return err
		}
		if err := packUint32(w, values[uint16(key)]); err != nil {
			return err
		}
	}
	return nil
}
