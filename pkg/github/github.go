package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ecksun/diffline/pkg/common"
)

type PRResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ID      string
				BaseRef struct {
					Target struct {
						OID string
					}
				}
				HeadRef struct {
					Target struct {
						OID string
					}
				}
				Reviews struct {
					Nodes []struct {
						ID       string
						BodyText string
						Comments struct {
							PageInfo struct {
								EndCursor string
							}
							Nodes []PRComment
						}
					}
				}
			}
		}
	}
}

type PRComment struct {
	Path     string
	Position int
	BodyText string
	Author   struct {
		Login string
	}
}

type PullRequest struct {
	ID      string
	BaseRef string
	HeadRef string
	Reviews []PullRequestReview
}

type PullRequestReview struct {
	ID       string
	Body     string
	Comments []PullRequestComment
}

type PullRequestComment struct {
	Path     string
	Position int
	Author   string
	Body     string
}

func GraphQLMustParse(name string, tmplStr string, data interface{}) []byte {
	buf := &bytes.Buffer{}

	tmpl := template.Must(template.New(name).Parse(tmplStr))

	tmpl.Execute(buf, data)
	query, err := json.Marshal(struct {
		Query string `json:"query"`
	}{
		Query: buf.String(),
	})
	if err != nil {
		panic(fmt.Errorf("failed to create JSON: %v", err))
	}
	return query
}

func simplifyPRStruct(pr PRResponse) PullRequest {
	var reviews []PullRequestReview
	for _, review := range pr.Data.Repository.PullRequest.Reviews.Nodes {
		var comments []PullRequestComment
		for _, comment := range review.Comments.Nodes {
			comments = append(comments, PullRequestComment{
				Path:     comment.Path,
				Position: comment.Position,
				Author:   comment.Author.Login,
				Body:     comment.BodyText,
			})
		}
		reviews = append(reviews, PullRequestReview{
			ID:       review.ID,
			Body:     review.BodyText,
			Comments: comments,
		})
	}

	return PullRequest{
		ID:      pr.Data.Repository.PullRequest.ID,
		BaseRef: pr.Data.Repository.PullRequest.BaseRef.Target.OID,
		HeadRef: pr.Data.Repository.PullRequest.HeadRef.Target.OID,
		Reviews: reviews,
	}
}

func ParsePRResponse(rawBody io.Reader) (PullRequest, error) {
	body, err := ioutil.ReadAll(rawBody)
	if err != nil {
		return PullRequest{}, err
	}

	var result PRResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return PullRequest{}, err
	}

	pr := simplifyPRStruct(result)
	return pr, nil
}

const (
	getPrGraphql string = `
{
  repository(owner: "{{ .Owner }}", name: "{{ .Repo }}") {
    pullRequest(number: {{ .PR }}) {
      id
      baseRef {
        target {
          oid
        }
      }
      headRef {
        target {
          oid
        }
      }
      reviews(author: "ecksun", first: 100) {
        nodes {
          id
          bodyText
          comments(first: 100) {
            pageInfo {
              endCursor
            }
            nodes {
              path
              position
              author {
                login
              }
              bodyText
            }
          }
        }
      }
    }
  }
}
`

	addPullRequestReviewGraphql string = `
mutation {
  addPullRequestReview(input: {
    body: "{{ .Body }}"
    pullRequestId: "{{ .PR }}"
    event: COMMENT
    comments: [
	{{range .Comments }}
	{
		path: "{{ .Path }}"
		position: {{ .Position }}
		body: "{{ .Body }}"
	}
	{{ end }}
	]
  }) {
    clientMutationId
  }
}
`
)

func CreateReview(pr string, body string, comments []common.GraphQLComment) error {
	query := GraphQLMustParse("add-review", addPullRequestReviewGraphql, struct {
		PR       string
		Body     string
		Comments []common.GraphQLComment
	}{
		PR:       pr,
		Body:     body,
		Comments: comments,
	})

	// reader, err := os.Open("./get-pr-diff.graphql")
	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer(query))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "bearer "+getGithubToken())

	client := &http.Client{}
	if _, err := client.Do(req); err != nil {
		return err
	}
	return nil
}

func GetPR(owner string, repo string, pr string) (PullRequest, error) {
	query := GraphQLMustParse("get-pr", getPrGraphql, struct {
		Owner string
		Repo  string
		PR    string
	}{
		Owner: owner,
		Repo:  repo,
		PR:    pr,
	})

	// reader, err := os.Open("./get-pr-diff.graphql")
	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer(query))
	if err != nil {
		return PullRequest{}, fmt.Errorf("could not created http request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "bearer "+getGithubToken())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return PullRequest{}, fmt.Errorf("got error response from client request: %v", err)
	}

	// TODO Handle error response
	return ParsePRResponse(res.Body)
}

const updateReviewGraphql = `
mutation {
  updatePullRequestReview(input: {
    pullRequestReviewId: "{{ .ID }}"
	body: "{{ .Body }}"
  }) {
    clientMutationId
  }
}
`

func getGithubToken() string {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr, "missing/empty GITHUB_TOKEN environment variable\n")
		os.Exit(1)
	}
	return token
}

func UpdateReview(reviewID string, body string) error {
	query := GraphQLMustParse("update-review", updateReviewGraphql, struct {
		ID   string
		Body string
	}{
		ID:   reviewID,
		Body: body,
	})

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer(query))
	if err != nil {
		return fmt.Errorf("failed to created update request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "bearer "+getGithubToken())

	client := &http.Client{}

	if _, err := client.Do(req); err != nil {
		return err
	}
	return nil
}
