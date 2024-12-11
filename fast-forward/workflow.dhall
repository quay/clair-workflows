let GithubActions =
      https://regadas.dev/github-actions-dhall/package.dhall
        sha256:dfb18adac8746b64040c5387d51769177ce08a2d2a496da2446eb244f34cc21e

let workflow =
          ../workflow-defaults.dhall
      /\  { name = "Fast Forward"
          , on = GithubActions.On::{
            , pull_request_review = Some GithubActions.PullRequestReview::{
              , types = Some [ "submitted" ]
              }
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
