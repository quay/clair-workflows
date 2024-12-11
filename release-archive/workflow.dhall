let GithubActions =
      https://regadas.dev/github-actions-dhall/package.dhall
        sha256:dfb18adac8746b64040c5387d51769177ce08a2d2a496da2446eb244f34cc21e

let lib = ../lib.dhall

let pairs-for = ./pairs-for.dhall

let InputType = GithubActions.types.InputType

let after-checkout = lib.steps.fn.after-checkout

let --| All the scripts used in this workflow.
    script =
      { --| Bash for fixing the checkout.
        fixtag = ./fixtag.sh as Text
      , --| Bash for invoking the archive-creation via the Makefile.
        make = ./make.sh as Text
      , --| Bash for reporting if there are commands in the project.
        check-cmds = ./checkcmds.sh as Text
      , --| Bash for generic go archive creation.
        go-project = ./goproject.sh as Text
      , --| TODO(hank) need to write this script
        todo = ./todo.sh as Text
      }

let variants =
      toMap
        { make =
          { run = script.make, want = [ "Makefile" ], nowant = [] : List Text }
        , go =
          { run = script.go-project
          , want = [ "go.mod" ]
          , nowant = [ "Makefile" ]
          }
        , other =
          { run = script.todo
          , want = [] : List Text
          , nowant = [ "Makefile", "go.mod" ]
          }
        }

let steps = pairs-for variants

let uploadExpr =
    --| Create a GitHub Expression that depends on the provided step IDs concluding "success".
      \(ids : List Text) ->
        let concat =
              \(it : Text) ->
              \(prev : Text) ->
                "steps.${it}.conclusion == 'success' || ${prev}"

        let expr = List/fold Text ids Text concat "false"

        in  "(${expr})"

let inputs =
      toMap
        { artifact-name =
          { default = Some "archive"
          , description = Some "Workflow artifact name to use"
          , required = False
          , type = Some InputType.string
          }
        }

let steps =
        [ lib.steps.checkout
        ,     GithubActions.steps.run { run = script.fixtag }
          //  { `if` = Some (after-checkout ([] : List Text)) }
        , lib.steps.fn.setup-go True
        ]
      # steps.steps
      # [ GithubActions.Step::{
          , name = Some "Check for Commands"
          , `if` = Some (after-checkout [ uploadExpr steps.upload-ids ])
          , id = Some "check-cmds"
          , run = Some script.check-cmds
          }
        ]

let w =
          ../workflow-defaults.dhall
      /\  { name = "Create Release Archive"
          , on = GithubActions.On::{
            , workflow_call = Some GithubActions.WorkflowCall::{
              , inputs = Some inputs
              , outputs = Some
                  ( toMap
                      { has-cmds =
                        { description = Some
                            ''
                            Set non-zero if there are directories under a directory named 'cmd'.
                            'Internal' directories are ignored.
                            ''
                        , value = "\${{ jobs.check-cmds.outputs.has-cmds }}"
                        }
                      }
                  )
              }
            }
          , jobs = toMap
              { lints = GithubActions.Job::{
                , name = Some "Archive"
                , runs-on = GithubActions.types.RunsOn.ubuntu-latest
                , steps
                }
              }
          }

in  GithubActions.Workflow::w
