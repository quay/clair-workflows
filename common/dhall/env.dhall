{-|
Env is helpers for GitHub Actions steps' environment.
-}
let List/map = https://prelude.dhall-lang.org/v22.0.0/List/map.dhall

let Map/Entry = https://prelude.dhall-lang.org/v22.0.0/Map/Entry.dhall

let Map/Type = https://prelude.dhall-lang.org/v22.0.0/Map/Type.dhall

let Map/keys = https://prelude.dhall-lang.org/v22.0.0/Map/keys.dhall

let T = https://prelude.dhall-lang.org/v22.0.0/Text/package.dhall

let GithubActions = https://regadas.dev/github-actions-dhall/package.dhall

let I = GithubActions.types.Input

let --| `InputsEnv` turns `workflow_call` inputs into a map suitable for a step's `env` key.
    inputsEnv
    : Map/Type Text I -> Map/Type Text Text
    = \(ins : Map/Type Text I) ->
        let keys = Map/keys Text I ins

        let --| Turn a key name into an environment variable name.
            envKey
            : Text -> Text
            = \(k : Text) -> "INPUT_" ++ T.replace "-" "_" (T.upperASCII k)

        let toEntry =
              \(e : Text) ->
                { mapKey = envKey e, mapValue = "\${{ inputs.${e} }}" }

        in  List/map Text (Map/Entry Text Text) toEntry keys

in  { inputsEnv }
