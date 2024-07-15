{-|
Steps is a bunch of helpers for GitHub Actions steps.
-}
let Map/map = https://prelude.dhall-lang.org/v22.0.0/Map/map.dhall

let T = https://prelude.dhall-lang.org/v22.0.0/Text/package.dhall

let GithubActions = https://regadas.dev/github-actions-dhall/package.dhall

let --| Generated via Makefile
    actions-versions =
      ./actions-versions_generated.dhall

let --| And-all joins expressions with ` && `
    and-all
    : List Text -> Text
    = \(exprs : List Text) -> T.concatSep " && " exprs

let --| Constructs a GitHub Expression suitable for after a "checkout" step.
    after-checkout
    : List Text -> Text
    = \(xs : List Text) ->
        let exprs = [ "!cancelled()", "steps.checkout.conclusion == 'success'" ]

        in  and-all (exprs # xs)

let --| Create a Step to use the `actions/setup-go` action, optionally using the cache.
    setup-go
    : Bool -> GithubActions.types.Step
    = \(cache : Bool) ->
        GithubActions.Step::{
        , name = Some "Setup Go"
        , id = Some "setupgo"
        , `if` = Some (after-checkout [ "hashFiles('go.mod') != ''" ])
        , uses = Some actions-versions.setup-go
        , `with` = Some
            ( toMap
                { cache = if cache then "true" else "false"
                , go-version = "stable"
                }
            )
        }

let --| Constants for the cache actions.
    cache =
      let prefix = "integration-assets-"

      in  { prefix, key = prefix ++ "\${{ hashFiles('go.sum') }}" }

let --| Common args for these `test-assets` Steps.
    assets-args =
      toMap
        { key = cache.key
        , restore-keys = T.concatSep "\n" [ cache.prefix ]
        , path = T.concatSep "\n" [ "~/.cache/clair-testing" ]
        }

let {-|
    `Test-assets-cache` restores the testing asset cache if available,
    and saves it back if the run is sucessful and the cache restore missed.
    -}
    test-assets-cache =
          GithubActions.Step
      //  { defaults =
            { name = Some "Assets Cache"
            , id = Some "assets"
            , `if` = Some "!cancelled()"
            , uses = Some actions-versions.cache
            , `with` = Some assets-args
            }
          }

let {-|
    `Test-assets-cache-save` saves the testing asset cache.

    Must be used with `test-assets-cache-restore`.
    -}
    test-assets-cache-save =
      let swapKey
          --| Replace the cache key with an expression to use the key recorded from `restore-assets`.
          : Text -> Text
          = \(v : Text) ->
              T.replace
                cache.key
                "\${{ steps.restore-assets.outputs.cache-primary-key }}"
                v

      in      GithubActions.Step
          //  { defaults =
                { name = Some "Assets Cache (Save)"
                , id = Some "save-assets"
                , `if` = Some "!cancelled()"
                , uses = Some actions-versions.cache/save
                , `with` = Some (Map/map Text Text Text swapKey assets-args)
                }
              }

let {-|
    `Test-assets-cache-restore` restores the testing assect cache.

    To save the modified cache back, use `test-assets-cache-save`.
    -}
    test-assets-cache-restore =
          GithubActions.Step
      //  { defaults =
            { name = Some "Assets Cache (Restore)"
            , id = Some "restore-assets"
            , `if` = Some "!cancelled()"
            , uses = Some actions-versions.cache/restore
            , `with` = Some assets-args
            }
          }

in  { --| Standard checkout step.
      checkout = GithubActions.Step::{
      , name = Some "Checkout"
      , id = Some "checkout"
      , `if` = Some "!cancelled()"
      , uses = Some actions-versions.checkout
      }
    , --| Helper functions
      fn =
      { setup-go, and-all, after-checkout }
    , --| Record construction helpers. Should be used with `::`.
      rec =
      { test-assets-cache, test-assets-cache-save, test-assets-cache-restore }
    }
