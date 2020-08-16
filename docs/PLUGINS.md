# Lighthouse plugins documentation

## Lighthouse plugins list

| plugin name           | configuration stanza      | docs |
| --------------------- | ------------------------- | ---- |
| approve               | `approve`                 | TODO |
| assign                |                           | TODO |
| blockade              | `blockades`               | TODO |
| branchcleaner         |                           | TODO |
| cat                   | `cat`                     | TODO |
| cherrypickunapproved  | `cherry_pick_unapproved`  | TODO |
| dog                   |                           | TODO |
| help                  |                           | TODO |
| hold                  |                           | [docs](./plugins/hold.md) |
| label                 | `label`                   | TODO |
| lgtm                  | `lgtm`                    | TODO |
| lifecycle             |                           | TODO |
| milestone             |                           | TODO |
| milestonestatus       |                           | TODO |
| override              |                           | TODO |
| owners-label          |                           | TODO |
| pony                  |                           | TODO |
| shrug                 |                           | [docs](./plugins/shrug.md) |
| sigmention            | `sigmention`              | TODO |
| size                  | `size`                    | [docs](./plugins/size.md) |
| skip                  |                           | TODO |
| stage                 |                           | TODO |
| trigger               | `triggers`                | TODO |
| updateconfig          | `config_updater`          | TODO |
| welcome               | `welcome`                 | [docs](./plugins/welcome.md) |
| wip                   |                           | [docs](./plugins/wip.md)  |
| yuks                  |                           | [docs](./plugins/yuks.md) |

## Plugins configuration file (plugins.yaml)

The _plugins.yaml_ file contains the configuration of all plugins (one stanza per plugin), and a map containing the list of plugins enabled per SCM repository.

You can lookup the stanza used to configure each plugin from the list above, and navigate to plugins documentation to find out the expected configuration structure.

Note that some plugins don't require any configuration and therefore do not have a configuration stanza.

The association between SCM repositories and their associated plugins lies in the `plugins` stanza. See an example below.

***plugins.yaml file structure:***
```yaml
# plugins configuration stanzas
approve: []
blockades: []
cat: {}
cherry_pick_unapproved: {}
config_updater: {}
heart: {}
label: {}
lgtm: []
repo_milestone: {}
require_matching_label: {}
requiresig: {}
sigmention: {}
size: {}
triggers: []
welcome: []

# external plugins configuration stanza
external_plugins: {}

# configuration related to handling OWNERS files
owners: {}

# repositories <-> plugins association stanza
plugins:
  # below is the list of plugins enabled for the org/repo repository
  # you can add/remove plugins from the list to enable/disable them for
  # the org/repo repository
  org/repo:
    - approve
    - assign
    - cat
    - dog
    - help
    - hold
    - label
    - lgtm
    - lifecycle
    - override
    - pony
    - shrug
    - size
    - skip
    - trigger
    - wip
    - yuks
```