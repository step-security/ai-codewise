package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type ApiClient struct {
	Client            *http.Client
	ApiBaseURI        string
	RepoName          string
	GitHubAccountName string
}

type CodeReviewRequestResponse struct {
	FullRepoName string `json:"full_repo_name"`
	CodeReviewID string `json:"code_review_id"`
}
type CodeReviewCommentsResponse struct {
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
	WaitInSeconds   int    `json:"wait_in_seconds"`
	CommentsCreated bool   `json:"comments_created"`
}

func (apiclient *ApiClient) performRequest(method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if headers != nil {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}

	return apiclient.Client.Do(req)
}

func (apiclient *ApiClient) SubmitCodeReviewRequest(prDetails *PullRequestDetails) (*CodeReviewRequestResponse, error) {
	url := fmt.Sprintf("%s/codereview/submit", apiclient.ApiBaseURI)
	jsonData, _ := json.Marshal(prDetails)

	headers := map[string]string{
		"Content-Type": "application/json; charset=UTF-8",
	}

	resp, err := apiclient.performRequest("POST", url, headers, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var codeReviewRequestResponse CodeReviewRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&codeReviewRequestResponse)
	if err != nil {
		return nil, err
	}

	return &codeReviewRequestResponse, nil
}

func (apiclient *ApiClient) GetCodeReviewComments(request *CodeReviewRequestResponse) (*CodeReviewCommentsResponse, error) {
	url := fmt.Sprintf("%s/codereview/comments?fullreponame=%s&codereviewid=%s", apiclient.ApiBaseURI, request.FullRepoName, request.CodeReviewID)

	resp, err := apiclient.performRequest("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, message: %s", resp.StatusCode, string(responseBody))
	}

	var codeReviewCommentsResponse CodeReviewCommentsResponse
	err = json.Unmarshal(responseBody, &codeReviewCommentsResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return &codeReviewCommentsResponse, nil
}
