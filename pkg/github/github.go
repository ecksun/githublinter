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
	Body     string
	Author   struct {
		Login string
	}
}

type PullRequest struct {
	ID       string
	BaseRef  string
	HeadRef  string
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
		panic(err)
	}
	return query
}

func simplifyPRStruct(pr PRResponse) PullRequest {
	var comments []PullRequestComment
	for _, review := range pr.Data.Repository.PullRequest.Reviews.Nodes {
		for _, comment := range review.Comments.Nodes {
			comments = append(comments, PullRequestComment{
				Path:     comment.Path,
				Position: comment.Position,
				Author:   comment.Author.Login,
				Body:     comment.Body,
			})
		}
	}
	return PullRequest{
		ID:       pr.Data.Repository.PullRequest.ID,
		BaseRef:  pr.Data.Repository.PullRequest.BaseRef.Target.OID,
		HeadRef:  pr.Data.Repository.PullRequest.HeadRef.Target.OID,
		Comments: comments,
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
              body
            }
          }
        }
      }
    }
  }
}
`
)

func GetPR(owner string, repo string, pr string) (PullRequest, error) {
	// TODO Extract environment reading to main
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr, "missing/empty TOKEN environment variable")
		os.Exit(1)
	}

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
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "bearer "+token)

	client := &http.Client{}
	res, err := client.Do(req)

	return ParsePRResponse(res.Body)
}
