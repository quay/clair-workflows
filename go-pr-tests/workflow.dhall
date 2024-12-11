let GithubActions =
      https://regadas.dev/github-actions-dhall/package.dhall
        sha256:dfb18adac8746b64040c5387d51769177ce08a2d2a496da2446eb244f34cc21e

let lib = ../lib.dhall

let InputType = GithubActions.types.InputType

let inputs =
      toMap
        { coverage =
          { default = Some "false"
          , description = Some "tktk"
          , required = False
          , type = Some InputType.boolean
          }
        , race =
          { default = Some "false"
          , description = Some "tktk"
          , required = False
          , type = Some InputType.boolean
          }
        , build-tags =
          { default = None Text
          , description = Some "tktk"
          , required = False
          , type = Some InputType.string
          }
        , make-target =
          { default = Some "check"
          , description = Some "make target to be used, if a Makefile exists"
          , required = False
          , type = Some InputType.string
          }
        }

let env = Some (lib.env.inputsEnv inputs)

let workflow =
          ../workflow-defaults.dhall
      /\  { name = "Go Tests"
          , on = GithubActions.On::{
            , workflow_call = Some GithubActions.WorkflowCall::{
              , inputs = Some inputs
              }
            }
          , jobs = toMap
              { test = GithubActions.Job::{
                , name = Some "Tests"
                , runs-on = GithubActions.types.RunsOn.ubuntu-latest
                , strategy = Some GithubActions.Strategy::{
                  , matrix = toMap { go = lib.go-versions }
                  }
                , steps =
                  [ GithubActions.Step::{
                    , uses = Some lib.actions-versions.setup-go
                    , id = Some "go"
                    , `with` = Some (toMap { go-version = "\${{matrix.go}}" })
                    }
                  , GithubActions.Step::{
                    , name = Some "Go Environment"
                    , `if` = Some "env.RUNNER_DEBUG == 1"
                    , run = Some ./report-env.sh as Text
                    , env
                    }
                  , GithubActions.Step::{
                    , name = Some "Run Tests (Makefile)"
                    , id = Some "test-make"
                    , `if` = Some "hashFiles('Makefile') != ''"
                    , run = Some ./run-make.sh as Text
                    , env
                    }
                  , GithubActions.Step::{
                    , name = Some "Run Test (find)"
                    , id = Some "test-find"
                    , `if` = Some "hashFiles('Makefile') == ''"
                    , --| This script gets buckwild to be able to get coverage output.
                      run = Some
                        ./run-find.sh as Text
                    , env
                    }
                  , GithubActions.Step::{
                    , name = Some "Report Coverage"
                    , `if` = Some
                        (Text/replace "\n" " " ./coverage-conditional as Text)
                    , uses = Some lib.actions-versions.codecov-action
                    , `with` = Some
                        ( toMap
                            { directory =
                                "\${{ steps.test-make.outputs.coverdir || steps.test-find.outputs.coverdir }}"
                            , override-branch = "\${{ github.ref_name }}"
                            }
                        )
                    }
                  , GithubActions.Step::{
                    , name = Some "Database Logs"
                    , `if` = Some "failure() || env.RUNNER_DEBUG == 1"
                    , run = Some ./db-logs.sh as Text
                    }
                  ]
                }
              }
          }

in  GithubActions.Workflow::workflow
