{-|
`Lib` is bunch of helpers.
-}
{ {-|
  `Actions_versions` is a list of commonly used actions.

  Actions that are used in more than one place should be added to `./lib/actions-versions.dhall`.
  -}
  actions-versions = ./lib/actions-versions_generated.dhall
, --| Go versions to use throughout.
  go-versions =
  [ "oldstable", "stable" ]
, env = ./lib/env.dhall
, steps = ./lib/steps.dhall
}
