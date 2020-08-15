# welcome

`welcome` plugin documentation:
- [Description](#description)
- [Commands](#commands)
- [Configuration](#configuration)
- [Compatibility matrix](#compatibility-matrix)

## Description

The welcome plugin posts a welcoming message in the pull request comments when it detects a user's first contribution to a repo.

The welcoming message can be configured per SCM repository.

Looking up the welcoming message for a given repository is done in the following order:
- `org/repo` first
- `org` only if there was no `org/repo` match
- [default message template](#default-message-template) is used if there was no match

## Commands

This plugin has no commands.

## Configuration

### Configuration stanza

| stanza    | type                       |
| --------- | -------------------------- |
| `welcome` | [][Welcome](#welcome-type) |

### Welcome type

| field              | type     | note                                                                                                                                    |
| ------------------ | -------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `repos`            | []string | can be in the form `org/repo` or just `org`                                                                                             |
| `message_template` | string   | go template used to create the welcoming message, see [Infos provided to the message template](#infos-provided-to-the-message-template) |

### Infos provided to the message template

| key         | type   |
| ----------- | ------ |
| Org         | string |
| Repo        | string |
| AuthorLogin | string |
| AuthorName  | string |

### Default message template

"Welcome @{{.AuthorLogin}}! It looks like this is your first PR to {{.Org}}/{{.Repo}} 🎉"

### Example

```yaml
welcome:
  - repos:
      - org1/repo1
      - org1/repo2
    message_template: Welcome @{{.AuthorLogin}} !
  - repos:
      - org2
    message_template: Nice to meet you @{{.AuthorLogin}} !
```

## Compatibility matrix

|               | GitHub | GitHub Enterprise | BitBucket Server | GitLab |
| ------------- | ------ | ----------------- | ---------------- | ------ |
| Pull requests | Yes    | Yes               | Yes              | Yes    |
| Commits       | No     | No                | No               | No     |
