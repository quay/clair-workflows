let GithubActions =
      https://regadas.dev/github-actions-dhall/package.dhall
        sha256:56b2d746cf5bf75b66276f5adaa057201bbe1ebf29836f4e35390e2a2bb68965

let workflow =
          ../workflow-defaults.dhall
      /\  { name = "Fast Forward"
          , on = GithubActions.On::{
            , pull_request_review = Some GithubActions.PullRequestReview::{
              , types = Some [ "submitted" ]
              }
              , workflow_call = Some GithubActions.WorkflowCall::{=}
            }
          , jobs = toMap
              { fast-forward = GithubActions.Job::{
                , name = Some "Fast Forward"
                , runs-on = GithubActions.types.RunsOn.ubuntu-latest
                , `if` = Some ./condition as Text
                , steps =
                  [ GithubActions.Step::{
                    , name = Some "Fast Forwarding"
                    , uses = Some "sequoia-pgp/fast-forward@v1"
                    , `with` = Some
                      [ { mapKey = "comment", mapValue = "on-error" }
                      , { mapKey = "merge", mapValue = "true" }
                      ]
                    }
                  ]
                }
              }
          }

in  GithubActions.Workflow::workflow
