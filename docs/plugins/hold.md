# hold

`hold` plugin documentation:
- [Description](#description)
- [Commands](#commands)
- [Configuration](#configuration)
- [Compatibility matrix](#compatibility-matrix)

## Description

The hold plugin allows anyone to add or remove the `do-not-merge/hold` Label from a pull request.

This label is typically used to temporarily prevent the pull request from merging without withholding approval.

## Commands

### /hold or /lh-hold

The `/hold` or `/lh-hold` commands add the `do-not-merge/hold` label to a pull request.

### /hold cancel or /lh-hold cancel

The `/hold cancel` or `/lh-hold cancel` commands remove the `do-not-merge/hold` label to a pull request.

## Configuration

This plugin has no configuration option.

## Compatibility matrix

|               | GitHub | GitHub Enterprise | BitBucket Server | GitLab |
| ------------- | ------ | ----------------- | ---------------- | ------ |
| Pull requests | Yes    | Yes               | Yes              | Yes    |
| Commits       | No     | No                | No               | No     |
