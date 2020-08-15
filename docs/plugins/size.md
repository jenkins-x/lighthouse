# yuks

`size` plugin documentation:
- [Description](#description)
- [Commands](#commands)
- [Configuration](#configuration)
- [Compatibility matrix](#compatibility-matrix)

## Description

The size plugin manages the `size/*` labels of pull requests, maintaining the appropriate label on each pull request as it is updated.

Generated files identified by the config file `.generated_files` at the repository root are ignored.

Labels are applied based on the total number of lines of changes (additions and deletions).

Thresholds for `XL`, `S`, `M`, `L`, `XL` and `XXL` sizes can be [configured](#confiiguration), if not configured [default size thresholds](#default-size-thresholds) are used.

## Commands

This plugin has no commands.

## Configuration

### Configuration stanza

| stanza    | type                       |
| --------- | -------------------------- |
| `size`    | [Size](#size-type)         |

### Size type

| field   | type     | note                                                 | default value |
| ------- | -------- | ---------------------------------------------------- | ------------- |
| `s`     | int      | number of lines of changes to apply the `s` size     | 10            |
| `m`     | int      | number of lines of changes to apply the `m` size     | 30            |
| `l`     | int      | number of lines of changes to apply the `l` size     | 100           |
| `xl`    | int      | number of lines of changes to apply the `xl` size    | 500           |
| `xxl`   | int      | number of lines of changes to apply the `xxl` size   | 1000          |

### Default size thresholds

| size         | threshold |
| ------------ | --------- |
| `size/XS`    | 0 - 9     |
| `size/S`     | 10 - 29   |
| `size/M`     | 30 - 99   |
| `size/L`     | 100 - 499 |
| `size/XL`    | 500 - 999 |
| `size/XXL`   | 1000+     |

### Example

```yaml
size:
  s: 20
  m: 50
  l: 150
  xl: 800
  xxl: 1500
```

## Compatibility matrix

|               | GitHub | GitHub Enterprise | BitBucket Server | GitLab |
| ------------- | ------ | ----------------- | ---------------- | ------ |
| Pull requests | Yes    | Yes               | Yes              | Yes    |
| Commits       | No     | No                | No               | No     |
