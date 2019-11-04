#!/bin/bash

set -uo pipefail

github_url_regex=".*//github.com/([^/]+)/([^/]+)/pull/([0-9]+)"

if [[ $1 =~ $github_url_regex ]]; then
    owner="${BASH_REMATCH[1]}"
    repo="${BASH_REMATCH[2]}"
    GITHUB_PR_NUMBER="${BASH_REMATCH[3]}"
else
    echo >&2 "Could not parse URL :("
    exit 1
fi

github_graphql() {
    echo >&2 "Running query: $1"
    query="$(tr -d '\n' <<< "$1")"
    curl \
        --silent \
        --header "Authorization: bearer $TOKEN" \
        -X POST \
        --data "$(jq --null-input --arg query "$query" '{ query: $query }')" \
        https://api.github.com/graphql
}

graphql="$(m4 \
    --define "PR_ID=$GITHUB_PR_NUMBER" \
    --define "OWNER=$owner" \
    --define "REPO=$repo" \
    get-pr-diff.graphql)"

TMPDIR=$(mktemp --directory --suffix=-diffline)
trap 'rm -rf $TMPDIR' EXIT

github_graphql "$graphql" > "$TMPDIR/api.json"

base=$(jq --raw-output '.data.repository.pullRequest.baseRef.target.oid' < "$TMPDIR/api.json")
head=$(jq --raw-output '.data.repository.pullRequest.headRef.target.oid' < "$TMPDIR/api.json")
pr_id=$(jq --raw-output '.data.repository.pullRequest.id' < "$TMPDIR/api.json")

(cd "$GIT_REPO" || exit 1
git diff "$base" "$head" > "$TMPDIR/diff"
)

mkdir -p "$TMPDIR/vet"
(cd "$GIT_REPO" && go vet > "$TMPDIR/vet/go_vet" 2>&1 )
(cd "$GIT_REPO" && go run github.com/alexkohler/nakedret ./... > "$TMPDIR/vet/nakedret" 2>&1 )
(cd "$GIT_REPO" && go run github.com/alexkohler/unimport ./... > "$TMPDIR/vet/unimport" 2>&1 )

comment_query=$(./githublinter "$TMPDIR/diff" <(find "$TMPDIR/vet/" -type f -print0 | xargs -0 cat))

mutation=$(m4 \
    --define "PR_ID=$pr_id" \
    --define "PR_COMMENTS=$comment_query" \
    comment-pr.graphql
)

github_graphql "$mutation"
