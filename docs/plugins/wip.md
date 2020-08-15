# wip

`wip` plugin documentation:
- [Description](#description)
- [Commands](#commands)
- [Configuration](#configuration)
- [Compatibility matrix](#compatibility-matrix)

## Description

The wip (work in progress) plugin applies the `do-not-merge/work-in-progress` label to pull requests.

Pull requests whose title starts with 'WIP' or are in the 'Draft' stage also get the `do-not-merge/work-in-progress` label applied.
The label is removed when the 'WIP' title prefix is removed or the pull request becomes ready for review.
 
The `do-not-merge/work-in-progress` label is typically used to block a pull request from merging while it is still in progress.

## Commands

This plugin has no commands.

## Configuration

This plugin has no configuration option.

## Compatibility matrix

|               | GitHub | GitHub Enterprise | BitBucket Server | GitLab |
| ------------- | ------ | ----------------- | ---------------- | ------ |
| Pull requests | Yes    | Yes               | Yes              | Yes    |
| Commits       | No     | No                | No               | No     |
