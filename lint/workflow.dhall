--| The "Lint" workflow.
let GithubActions =
      https://regadas.dev/github-actions-dhall/package.dhall
        sha256:dfb18adac8746b64040c5387d51769177ce08a2d2a496da2446eb244f34cc21e

let lib = ../lib.dhall

let InputType = GithubActions.types.InputType

let script =
      { --| GithubScript for checking commits.
        commits = ./script.js as Text
      , --| Bash for checking an `openapi.yaml` file.
        api = ./check-api.sh as Text
      , --| Bash for checking versions in `Dockerfile` and `docker-compose.yaml` files.
        container = ./check-container.sh as Text
      , --| Bash for checking file names.
        filenames = ./check-filenames.sh as Text
      , --| Bash for checking spelling.
        misspell = ./check-spelling.sh as Text
      , --| Bash for checking any `go.mod` files.
        go-tidy = ./go-tidy.sh as Text
      }

let inputs =
      toMap
        { flags =
          { default = Some "gm"
          , description = Some "Regexp flags"
          , required = False
          , type = Some InputType.string
          }
        , pattern =
          { default = Some
              ''
              ^[^:!]+: .+\n\n.*$
              ''
          , description = Some "Commit message pattern"
          , required = False
          , type = Some InputType.string
          }
        }

let after-checkout = \(xs : List Text) -> lib.steps.fn.after-checkout xs

let --| Need an explicit cache step because the `setup-go` action only uses a single cache key.
    steps =
      [ GithubActions.Step::{
        , name = Some "Commit Check"
        , env = Some (lib.env.inputsEnv inputs)
        , uses = Some lib.actions-versions.github-script
        , `with` = Some (toMap { script = script.commits })
        }
      , lib.steps.checkout
      , GithubActions.Step::{
        , id = Some "filenames"
        , name = Some "Check Filenames"
        , `if` = Some (after-checkout ([] : List Text))
        , run = Some script.filenames
        }
      , GithubActions.Step::{
        , id = Some "api"
        , name = Some "Check API Reference"
        , `if` = Some (after-checkout [ "hashFiles('openapi.yaml') != ''" ])
        , run = Some script.api
        }
      , GithubActions.Step::{
        , id = Some "container"
        , name = Some "Check Container Versions"
        , `if` = Some (after-checkout ([] : List Text))
        , run = Some script.container
        }
      , GithubActions.Step::{
        , name = Some "Setup Go"
        , id = Some "setupgo"
        , `if` = Some (after-checkout ([] : List Text))
        , uses = Some lib.actions-versions.setup-go
        , `with` = Some (toMap { cache = "false", go-version = "stable" })
        }
      , GithubActions.Step::{
        , name = Some "Setup Tool Cache"
        , id = Some "toolcache"
        , `if` = Some
            (after-checkout [ "steps.setupgo.conclusion == 'success'" ])
        , uses = Some lib.actions-versions.cache
        , `with` = Some
            ( toMap
                { key = "lint-\${{ github.action }}-\${{ github.workflow_sha }}"
                , restore-keys = "lint-\${{ github.action }}-"
                , paths =
                    ''
                    ~/.cache/go-build
                    ~/go/pkg/mod
                    ''
                }
            )
        }
      , GithubActions.Step::{
        , id = Some "tidy"
        , name = Some "Go Tidy"
        , `if` = Some
            ( after-checkout
                [ "steps.setupgo.conclusion == 'success'"
                , "hashFiles('**/go.mod') != ''"
                ]
            )
        , run = Some script.go-tidy
        }
      , GithubActions.Step::{
        , id = Some "misspell"
        , name = Some "Spellcheck"
        , `if` = Some
            (after-checkout [ "steps.setupgo.conclusion == 'success'" ])
        , run = Some script.misspell
        }
      ]

let w =
          ../workflow-defaults.dhall
      /\  { name = "Lints"
          , on = GithubActions.On::{
            , workflow_call = Some GithubActions.WorkflowCall::{
              , inputs = Some inputs
              }
            }
          , jobs = toMap
              { lints = GithubActions.Job::{
                , name = Some "Lints"
                , runs-on = GithubActions.types.RunsOn.ubuntu-latest
                , steps
                }
              }
          }

in  GithubActions.Workflow::w
