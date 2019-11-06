# Lint github pull-requests

Say you have a new lint-rule you would like to introduce but turning on that
linting for everything is infeasable because of the number of them. This tool
allows you to only report linting issues in changes made by pull-requsts. That
way you can ensure that the number of issues are decreasing over time without
having to fix all instance of the issue at once.

## Usage

```bash
export GITHUB_TOKEN="..."
export GIT_REPO=. # the path to the git repo the PR is part of
githublinter https://github.com/ecksun/test-repo/5 ./lint.sh
```

Currently there is special syntax used by `githublinter` to annotate lints with
metadata, for example if you are running multiple linters at once and want to
have the name of the linter as part of the message in the pull-request.

That syntax is `# githublinter: ${message}`, for example you might have a lint
file that looks like this:

```bash
#!/bin/sh

echo '# githublinter: go vet'
go vet 2>&1
echo '# githublinter: nakedret'
nakedret
echo '# githublinter: unimport'
unimport
```
