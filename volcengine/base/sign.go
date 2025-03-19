package base

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Credentials contains the credentials
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	Service         string
	Region          string
	SessionToken    string
}

// Sign signs the request with the given credentials
func Sign(credentials Credentials, request *http.Request) {
	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" {
		return
	}

	t := time.Now().UTC()

	date := t.Format("20060102")
	amzdate := t.Format("20060102T150405Z")

	request.Header.Set("X-Date", amzdate)

	if credentials.SessionToken != "" {
		request.Header.Set("X-Security-Token", credentials.SessionToken)
	}

	host := request.URL.Host
	if !strings.Contains(host, ":") && request.URL.Port() != "" {
		host = fmt.Sprintf("%s:%s", host, request.URL.Port())
	}
	request.Header.Set("Host", host)

	canonicalURI := request.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQueryString := request.URL.RawQuery

	canonicalHeaders := fmt.Sprintf("host:%s\nx-date:%s\n", host, amzdate)

	signedHeaders := "host;x-date"

	if credentials.SessionToken != "" {
		canonicalHeaders += fmt.Sprintf("x-security-token:%s\n", credentials.SessionToken)
		signedHeaders += ";x-security-token"
	}

	payloadHash := hashSHA256([]byte(""))

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		request.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	credentialScope := fmt.Sprintf("%s/%s/%s/request", date, credentials.Region, credentials.Service)

	stringToSign := fmt.Sprintf("HMAC-SHA256\n%s\n%s\n%s",
		amzdate,
		credentialScope,
		hashSHA256([]byte(canonicalRequest)),
	)

	signingKey := getSignatureKey(credentials.SecretAccessKey, date, credentials.Region, credentials.Service)
	signature := hmacSHA256(signingKey, stringToSign)

	authorizationHeader := fmt.Sprintf("HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		credentials.AccessKeyID,
		credentialScope,
		signedHeaders,
		hex.EncodeToString(signature),
	)

	request.Header.Set("Authorization", authorizationHeader)
}

func hashSHA256(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func hmacSHA256(key []byte, data string) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(data))
	return hash.Sum(nil)
}

func getSignatureKey(key, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("VOLC"+key), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "request")
	return kSigning
}

// SignUrl signs the url with the given credentials
func SignUrl(credentials Credentials, method string, rawUrl string, body string, timeout time.Duration) (string, error) {
	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" {
		return rawUrl, nil
	}

	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse url: %v", err)
	}

	query := parsedUrl.Query()

	t := time.Now().UTC()

	date := t.Format("20060102")
	amzdate := t.Format("20060102T150405Z")

	query.Set("X-Date", amzdate)

	if credentials.SessionToken != "" {
		query.Set("X-Security-Token", credentials.SessionToken)
	}

	if timeout > 0 {
		query.Set("X-Expires", fmt.Sprintf("%d", int(timeout.Seconds())))
	}

	host := parsedUrl.Host
	if !strings.Contains(host, ":") && parsedUrl.Port() != "" {
		host = fmt.Sprintf("%s:%s", host, parsedUrl.Port())
	}

	canonicalURI := parsedUrl.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(query))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(query.Get(k))))
	}
	canonicalQueryString := strings.Join(parts, "&")

	canonicalHeaders := fmt.Sprintf("host:%s\n", host)

	signedHeaders := "host"

	payloadHash := hashSHA256([]byte(body))

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	credentialScope := fmt.Sprintf("%s/%s/%s/request", date, credentials.Region, credentials.Service)

	stringToSign := fmt.Sprintf("HMAC-SHA256\n%s\n%s\n%s",
		amzdate,
		credentialScope,
		hashSHA256([]byte(canonicalRequest)),
	)

	signingKey := getSignatureKey(credentials.SecretAccessKey, date, credentials.Region, credentials.Service)
	signature := hmacSHA256(signingKey, stringToSign)

	query.Set("X-Credential", fmt.Sprintf("%s/%s", credentials.AccessKeyID, credentialScope))
	query.Set("X-SignedHeaders", signedHeaders)
	query.Set("X-Signature", hex.EncodeToString(signature))

	parsedUrl.RawQuery = query.Encode()

	return parsedUrl.String(), nil
}
