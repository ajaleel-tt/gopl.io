package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetIssue fetches a single issue from a repository.
func GetIssue(owner, repo string, number int) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", APIURL, owner, repo, number)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("get issue failed: %s", resp.Status)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return &issue, nil
}

// CreateIssue creates a new issue in a repository.
func CreateIssue(owner, repo, token string, req IssueRequest) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues", APIURL, owner, repo)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "token "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		return nil, fmt.Errorf("create issue failed: %s", resp.Status)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return &issue, nil
}

// UpdateIssue updates an existing issue in a repository.
func UpdateIssue(owner, repo, token string, number int, req IssueRequest) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", APIURL, owner, repo, number)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("PATCH", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "token "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("update issue failed: %s", resp.Status)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return &issue, nil
}

// CloseIssue closes an existing issue in a repository.
func CloseIssue(owner, repo, token string, number int) (*Issue, error) {
	return UpdateIssue(owner, repo, token, number, IssueRequest{State: "closed"})
}
