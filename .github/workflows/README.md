## House style

### Default shell should be `bash`

```
defaults:
  run:
    shell: bash
```

Explicitly requesting `bash` executes snippets with the `errexit`, `nounset`, and `pipefail` options.

### Jobs should run on `ubuntu-latest`

```
jobs:
  config:
    runs-on: 'ubuntu-latest'
```

Nothing in the workflows should be forward-incompatible.

### Step `if` statements should be space-collapsing multiline strings

```
jobs:
  job:
    steps:
      - run: true
        if: >-
          !cancelled() &&
          (
              github.event_name == 'push'
          ) ||
          true
```

Keeping this uniform makes it easier to modify and see the logic for complex statements.
Logical operators should be at the end of line.
Grouping parenthesis should be on their own lines.
