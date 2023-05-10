package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/sethvargo/go-githubactions"
	"golang.org/x/oauth2"
)

// To be set by the build workflow
var APIEndpoint string

type PullRequestFileChanges struct {
	File   string `json:"file"`
	Status string `json:"status"`
	Patch  string `json:"patch"`
}

type PullRequestDetails struct {
	GitHubAccountName string                   `json:"github_account_name"`
	RepositoryName    string                   `json:"repository_name"`
	PullNumber        int                      `json:"pull_number"`
	FileChanges       []PullRequestFileChanges `json:"file_changes"`
	PullRequestAuthor string                   `json:"pull_request_author"`
}

const (
	OperationStatusDispatched = "Dispatched"
	OperationStatusSucceeded  = "Succeeded"
	OperationStatusError      = "Error"
)

func getTokenRemainingValidity(timestamp interface{}) float64 {
	if validity, ok := timestamp.(float64); ok {
		tm := time.Unix(int64(validity), 0)
		remainder := time.Until(tm)
		return remainder.Seconds()
	}
	return 0
}

func getGitHubClient() (*github.Client, context.Context, error) {
	pat := os.Getenv("PAT")
	if len(pat) == 0 {
		return nil, nil, errors.New("a GitHub token must be passed as 'PAT' variable to the action")
	}
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)
	httpClient := oauth2.NewClient(ctx, tokenSource)

	return github.NewClient(httpClient), ctx, nil
}

func getDebugMode() bool {
	isDebugMode := false
	debugModeStr, exists := os.LookupEnv("DebugMode")

	if exists {
		debugMode, err := strconv.ParseBool(debugModeStr)
		if err == nil {
			isDebugMode = debugMode
		}
	}
	return isDebugMode
}

func printDebugMessageIfRequired(isDebugMode bool, format string, args ...any) {
	if isDebugMode {
		githubactions.Infof(format, args)
	}
}

func getPullRequestDetailsFromEnvironment(isDebugMode bool) (*PullRequestDetails, error) {
	gitHubRepository, exists := os.LookupEnv("GITHUB_REPOSITORY")

	if !exists {
		return nil, errors.New("could not read GITHUB_REPOSITORY environment variable")
	}

	gitHubRepositoryParts := strings.Split(gitHubRepository, "/")
	githubAccountName := gitHubRepositoryParts[0]
	repositoryName := gitHubRepositoryParts[1]

	gitHubEvent, exists := os.LookupEnv("GITHUB_EVENT_NAME")
	if !exists {
		return nil, errors.New("could not read GITHUB_EVENT_NAME environment variable")
	}

	if !strings.EqualFold(gitHubEvent, "pull_request") {
		return nil, errors.New("github event is not pull request")
	}
	gitHubRef, exists := os.LookupEnv("GITHUB_REF")
	if !exists {
		return nil, errors.New("could not read GITHUB_REF environment variable")
	}

	gitHubRefParts := strings.Split(gitHubRef, "/")
	if len(gitHubRefParts) != 4 {
		return nil, errors.New("environment variable GITHUB_REF is not in expected format")

	}

	pullNumber, err := strconv.Atoi(gitHubRefParts[2])
	if err != nil {
		return nil, fmt.Errorf("error converting pull request number %s to int. error:%v", gitHubRefParts[2], err)
	}

	client, ctx, err := getGitHubClient()
	if err != nil {
		return nil, err
	}
	pr, _, err := client.PullRequests.Get(ctx, githubAccountName, repositoryName, pullNumber)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request: %v", err)
	}

	// Get GitHub user who created this pull request
	pullRequestAuthor := pr.GetUser().GetLogin()

	printDebugMessageIfRequired(isDebugMode, "owner:%s repo=%s pullNumber=%d author=%s", githubAccountName, repositoryName, pullNumber, pullRequestAuthor)
	comparison, _, err := client.Repositories.CompareCommits(ctx, githubAccountName, repositoryName, pr.GetBase().GetSHA(), pr.GetHead().GetSHA(), &github.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("error comparing commits: %v", err)
	}

	prDetails := PullRequestDetails{
		GitHubAccountName: githubAccountName,
		RepositoryName:    repositoryName,
		PullNumber:        pullNumber,
		FileChanges:       []PullRequestFileChanges{},
		PullRequestAuthor: pullRequestAuthor,
	}

	// Print file changes
	for _, file := range comparison.Files {
		printDebugMessageIfRequired(isDebugMode, "File: %s, Status: %s Diff:\n%s\n", file.GetFilename(), file.GetStatus(), file.GetPatch())
		prDetails.FileChanges = append(prDetails.FileChanges, PullRequestFileChanges{
			File:   file.GetFilename(),
			Status: file.GetStatus(),
			Patch:  file.GetPatch(),
		})

	}
	return &prDetails, nil
}

func submitPRDetailsAndGetCodeFeedback(prDetails *PullRequestDetails, isDebugMode bool) (bool, error) {
	responseReceived := false
	audience := APIEndpoint
	oidcClient, err := DefaultOIDCClient(audience)
	if err != nil {
		return responseReceived, fmt.Errorf("error generating OIDC auth token. error:%v", err)
	}

	actionsJWT, exp, err := getActionsJWTAndExp(oidcClient, isDebugMode)
	if err != nil {
		return responseReceived, fmt.Errorf("error generating OIDC auth token. error:%v", err)
	}

	apiClient := ApiClient{
		Client:     &http.Client{},
		ApiBaseURI: APIEndpoint + "/v1/app/",
	}
	response, err := apiClient.SubmitCodeReviewRequest(actionsJWT.Value, prDetails)
	if err != nil {
		return responseReceived, fmt.Errorf("error submitting code review request: %v", err)
	}
	responseBytes, _ := json.Marshal(response)

	printDebugMessageIfRequired(isDebugMode, "SubmitCodeReviewResponse:%s", string(responseBytes))
	time.Sleep(20 * time.Second)
	var reviewComments *CodeReviewCommentsResponse

	for i := 0; i < 20 && !responseReceived; i++ {
		remainder := getTokenRemainingValidity(exp)
		if remainder < 60 {
			githubactions.Infof("Renewing OIDC token as it's only valid for %f", remainder)
			actionsJWT, exp, err = getActionsJWTAndExp(oidcClient, isDebugMode)
			if err != nil {
				return responseReceived, fmt.Errorf("error renewing OIDC token. Error: %v", err)
			}
		}
		reviewComments, err = apiClient.GetCodeReviewComments(actionsJWT.Value, response)
		if err != nil {
			return responseReceived, fmt.Errorf("error retrieving code review comments: %v", err)
		}
		if reviewComments.Status == OperationStatusDispatched {
			githubactions.Infof("%d attempt to retrieve response: sleeping for 30 seconds", i)
			time.Sleep(30 * time.Second)
		} else {
			responseReceived = true
			if reviewComments.Status == OperationStatusError {
				message := fmt.Sprintf("Error while using StepSecurity AI Code Reviewer. \nError details:%s", reviewComments.Error)
				client, ctx, err := getGitHubClient()
				if err != nil {
					return responseReceived, fmt.Errorf("error getting github client:%v", err)
				}
				comment := "COMMENT"
				_, commentResponse, err := client.PullRequests.CreateReview(
					ctx,
					prDetails.GitHubAccountName,
					prDetails.RepositoryName,
					prDetails.PullNumber,
					&github.PullRequestReviewRequest{
						Body:  &message,
						Event: &comment,
					})
				if err != nil {
					errorMessage := fmt.Sprintf("Error writing comment on pull request: %v\n", err)
					responseBody, err := ioutil.ReadAll(commentResponse.Body)
					if err == nil {
						errorMessage += fmt.Sprintf(" response body:%s", responseBody)
					} else {
						errorMessage += fmt.Sprintf(" could not retrieve response body for error details. error:%v", err)
					}
					return responseReceived, errors.New(errorMessage)
				}
			}
			break
		}
	}
	return responseReceived, nil
}

func main() {
	isDebugMode := getDebugMode()
	envVariables := strings.Join(os.Environ(), ",")
	printDebugMessageIfRequired(isDebugMode, "Environment Variables:%s", envVariables)

	prDetails, err := getPullRequestDetailsFromEnvironment(isDebugMode)
	if err != nil {
		githubactions.Errorf("could not retrieve pull request details. Error:%v", err)
		return
	}

	if strings.EqualFold(prDetails.PullRequestAuthor, "dependabot[bot]") || strings.EqualFold(prDetails.PullRequestAuthor, "renovate[bot]") {
		githubactions.Infof("Skipping as the PR is created by a dependency update bot (%s)", prDetails.PullRequestAuthor)
		return
	}

	responseReceived, err := submitPRDetailsAndGetCodeFeedback(prDetails, isDebugMode)
	if err != nil {
		githubactions.Errorf("error while processing pull request changes with StepSecurity APIs. Error details:%v", err)
		return
	}

	if !responseReceived {
		message := "StepSecurity AI Code Reviewer request timed out after 10 minutes"
		comment := "COMMENT"
		client, ctx, err := getGitHubClient()
		if err != nil {
			githubactions.Errorf("error getting github client:%v", err)
			return
		}
		client.PullRequests.CreateReview(
			ctx,
			prDetails.GitHubAccountName,
			prDetails.RepositoryName,
			prDetails.PullNumber,
			&github.PullRequestReviewRequest{
				Body:  &message,
				Event: &comment,
			})

		githubactions.Fatalf(message)
	}
}
