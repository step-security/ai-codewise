# AI-CodeWise

<p align="center">
  <img  src="images/banner.png" width="400">
</p>

<div align="center">

[![Maintained by stepsecurity.io](https://img.shields.io/badge/maintained%20by-stepsecurity.io-blueviolet)](https://stepsecurity.io/?utm_source=github&utm_medium=organic_oss&utm_campaign=harden-runner)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://raw.githubusercontent.com/step-security/harden-runner/main/LICENSE)

</div>

---

<p align="center">
AI-Powered Code Reviews for Best Practices & Security Issues Across Languages
</p>

AI-CodeWise GitHub Action is an AI Code Reviewer.

- It is triggered on a pull request and sends the diff of the code files to the StepSecurity API, which then employs prompt engineering to call the Azure OpenAI API to review the code.

- AI-CodeWise automatically adds a pull request comment using the StepSecurity bot account.

- The comment contains detailed information about the identified issues to improve code quality and address potential security vulnerabilities.

<p align="center">
  <img src="images/sequence-diagram.png" alt="Sequence diagram">
</p>

## Usage

To use AI-CodeWise, add this GitHub Actions workflow to your repositories

```yaml
name: Code Review
on:
  pull_request:
permissions:
  contents: read
jobs:
  code-review:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
      id-token: write
    steps:
      - name: Harden Runner
        uses: step-security/harden-runner@128a63446a954579617e875aaab7d2978154e969 # v2.4.0
        with:
          disable-sudo: true
          egress-policy: block
          allowed-endpoints: >
            api.github.com:443
            int.api.stepsecurity.io:443

      - name: Code Review
        uses: step-security/ai-codewise@int
```

When you create a pull request in the repository, the workflow will get triggered and add a pull request comment. Here is an screenshot of what the comment will look like:
<p align="center">
<img src="images/sample-code-comment.png" width="600">
</p>

## Comparison with existing SAST and IaC scanners

AI-CodeWise differentiates itself from existing rule-based scanners by offering the following advantages:

1. Comprehensive Code Review: AI-CodeWise is a single code reviewer tool that can detect code smells, best practice violations, and security issues. Furthermore, it does this across different programming languages, making it a versatile solution for your code review needs.

2. Unanticipated Issues Detection: By harnessing the power of AI, AI-CodeWise can identify potential problems that traditional rule-based systems may not have considered or anticipated. This ensures a more thorough code analysis and helps prevent potential vulnerabilities from slipping through the cracks.

3. Suggested Fixes: In addition to identifying issues, AI-CodeWise goes a step further by suggesting code changes to implement fixes directly in the PR comment. These actionable insights empower developers to resolve issues more efficiently, ultimately improving overall code quality and security.

## Examples

Here are a few example pull requests with PR comments from AI-CodeWise

1.
2.
3.

## Limitations

- AI-CodeWise will only review changes if the total number of file changes in a pull request is less than 10.
- AI-CodeWise will only review changes if the total characters in the changes are less than 20K.
- AI-CodeWise only works for public repositories as of now. To use on a private repository, please join the beta.