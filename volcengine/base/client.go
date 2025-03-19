package base

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// ServiceInfo contains the info of service
type ServiceInfo struct {
	Timeout     time.Duration
	Scheme      string
	Host        string
	Header      http.Header
	Credentials Credentials
}

// ApiInfo contains the api info
type ApiInfo struct {
	Method  string
	Path    string
	Query   url.Values
	Form    url.Values
	Header  http.Header
	Timeout *time.Duration
}

// Client is the base client
type Client struct {
	ServiceInfo *ServiceInfo
	ApiInfoList map[string]*ApiInfo
	httpClient  *http.Client
}

// CommonResponse is the common response of api
type CommonResponse struct {
	ResponseMetadata *ResponseMetadata `json:"ResponseMetadata"`
}

// ResponseMetadata is the metadata of response
type ResponseMetadata struct {
	RequestId string `json:"RequestId"`
	Action    string `json:"Action"`
	Version   string `json:"Version"`
	Service   string `json:"Service"`
	Region    string `json:"Region"`
	Error     *Error `json:"Error,omitempty"`
}

// Error is the error of response
type Error struct {
	CodeN   int    `json:"CodeN,omitempty"`
	Code    string `json:"Code,omitempty"`
	Message string `json:"Message,omitempty"`
}

// NewClient returns a new client
func NewClient(info *ServiceInfo, apiInfoList map[string]*ApiInfo) *Client {
	tr := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     60 * time.Second,
	}

	client := &Client{
		ServiceInfo: info,
		ApiInfoList: apiInfoList,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   info.Timeout,
		},
	}

	return client
}

// GetSignUrl returns the signed url
func (client *Client) GetSignUrl(api string, query url.Values) (string, error) {
	apiInfo := client.ApiInfoList[api]
	if apiInfo == nil {
		return "", fmt.Errorf("no such api: %s", api)
	}

	query = mergeQuery(query, apiInfo.Query)

	return client.ServiceInfo.Scheme + "://" + client.ServiceInfo.Host + apiInfo.Path + "?" + query.Encode(), nil
}

// Json sends a json request
func (client *Client) Json(api string, query url.Values, body string) ([]byte, int, error) {
	return client.CtxJson(nil, api, query, body)
}

// CtxJson sends a json request with context
func (client *Client) CtxJson(ctx interface{}, api string, query url.Values, body string) ([]byte, int, error) {
	apiInfo := client.ApiInfoList[api]
	if apiInfo == nil {
		return nil, 0, fmt.Errorf("no such api: %s", api)
	}

	query = mergeQuery(query, apiInfo.Query)

	form := mergeValues(url.Values{}, apiInfo.Form)

	method := apiInfo.Method
	path := apiInfo.Path

	var req *http.Request
	var err error
	if method == http.MethodGet {
		req, err = http.NewRequest(method, "", nil)
	} else {
		buf := bytes.NewBufferString(body)
		req, err = http.NewRequest(method, "", buf)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	scheme := client.ServiceInfo.Scheme
	host := client.ServiceInfo.Host

	req.URL = &url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: query.Encode(),
	}

	for key, values := range apiInfo.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	for key, values := range client.ServiceInfo.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	Sign(client.ServiceInfo.Credentials, req)

	httpClient := client.httpClient
	if apiInfo.Timeout != nil {
		httpClient = &http.Client{
			Transport: client.httpClient.Transport,
			Timeout:   *apiInfo.Timeout,
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to do request: %v", err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return data, resp.StatusCode, fmt.Errorf("http status code: %d, message: %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

// Query sends a query request
func (client *Client) Query(api string, query url.Values) ([]byte, int, error) {
	return client.CtxQuery(nil, api, query)
}

// CtxQuery sends a query request with context
func (client *Client) CtxQuery(ctx interface{}, api string, query url.Values) ([]byte, int, error) {
	return client.CtxJson(ctx, api, query, "")
}

// Post sends a post request
func (client *Client) Post(api string, query url.Values, form url.Values) ([]byte, int, error) {
	return client.CtxPost(nil, api, query, form)
}

// CtxPost sends a post request with context
func (client *Client) CtxPost(ctx interface{}, api string, query url.Values, form url.Values) ([]byte, int, error) {
	apiInfo := client.ApiInfoList[api]
	if apiInfo == nil {
		return nil, 0, fmt.Errorf("no such api: %s", api)
	}

	query = mergeQuery(query, apiInfo.Query)

	form = mergeValues(form, apiInfo.Form)

	method := apiInfo.Method
	path := apiInfo.Path

	var req *http.Request
	var err error
	if method == http.MethodGet {
		req, err = http.NewRequest(method, "", nil)
	} else {
		buf := bytes.NewBufferString(form.Encode())
		req, err = http.NewRequest(method, "", buf)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	scheme := client.ServiceInfo.Scheme
	host := client.ServiceInfo.Host

	req.URL = &url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: query.Encode(),
	}

	for key, values := range apiInfo.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	for key, values := range client.ServiceInfo.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	Sign(client.ServiceInfo.Credentials, req)

	httpClient := client.httpClient
	if apiInfo.Timeout != nil {
		httpClient = &http.Client{
			Transport: client.httpClient.Transport,
			Timeout:   *apiInfo.Timeout,
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to do request: %v", err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return data, resp.StatusCode, fmt.Errorf("http status code: %d, message: %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

func mergeQuery(query url.Values, apiQuery url.Values) url.Values {
	if query == nil {
		query = url.Values{}
	}

	if apiQuery != nil {
		for key, values := range apiQuery {
			for _, value := range values {
				query.Add(key, value)
			}
		}
	}

	return query
}

func mergeValues(form url.Values, apiForm url.Values) url.Values {
	if form == nil {
		form = url.Values{}
	}

	if apiForm != nil {
		for key, values := range apiForm {
			for _, value := range values {
				form.Add(key, value)
			}
		}
	}

	return form
}
