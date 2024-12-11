let List/map = https://prelude.dhall-lang.org/v22.0.0/List/map.dhall

let List/concat = https://prelude.dhall-lang.org/v22.0.0/List/concat.dhall

let Entry = https://prelude.dhall-lang.org/v22.0.0/Map/Entry.dhall

let Map/keys = https://prelude.dhall-lang.org/v22.0.0/Map/keys.dhall

let GithubActions = https://regadas.dev/github-actions-dhall/package.dhall

let lib = ../lib.dhall

let after-checkout = lib.steps.fn.after-checkout

let Step = GithubActions.types.Step

let StepList = List Step

let mapText = List/map Text Text

let
    --| CreateArgs are the arguments needed for one variant:
    --|  - run: Text containing the bash script to run.
    --|  - want: Files that must exist to trigger this step.
    --|  - nowant: Files that must not exist to trigger this step.
    CreateArgs =
      { run : Text, want : List Text, nowant : List Text }

let CreateEntry = Entry Text CreateArgs

let CreateMap = List CreateEntry

let --| Returns the ID for the "creation" step.
    create-id =
      \(id : Text) -> "create-archive-${id}"

let --| Returns the ID for the "upload" step.
    upload-id =
      \(id : Text) -> "upload-archive-${id}"

let upload-for
    --| Return an upload step for the given name.
    : Text -> Step
    = \(name : Text) ->
        let prev = create-id name

        in  GithubActions.Step::{
            , name = Some "Upload Archive (${name})"
            , id = Some (upload-id name)
            , uses = Some lib.actions-versions.upload-artifact
            , `if` = Some
                (after-checkout [ "steps.${prev}.outputs.name != ''" ])
            , `with` = Some
                ( toMap
                    { if-no-files-found = "error"
                    , name = "\${{ inputs.artifact-name }}"
                    , path = "\${{ steps.${prev}.outputs.name }}"
                    }
                )
            }

let create-for
    --| Return a creation step for the given name, using provided CreateArgs.
    : Text -> CreateArgs -> Step
    = \(name : Text) ->
      \(args : CreateArgs) ->
        let mkexpr =
              \(cond : Text) -> \(f : Text) -> "hashFiles('${f}') ${cond} ''"

        let want = mapText (mkexpr "!=") args.want

        let nowant = mapText (mkexpr "==") args.nowant

        in  GithubActions.Step::{
            , name = Some "Create Archive (${name})"
            , id = Some "create-archive-${name}"
            , `if` = Some (after-checkout (want # nowant))
            , run = Some args.run
            }

let pair-for
    --| Reutrn paired "create" and "upload" steps.
    : CreateEntry -> StepList
    = \(kv : CreateEntry) ->
        let name = kv.mapKey

        let args = kv.mapValue

        in  [ create-for name args, upload-for name ]

let pairs-for
    --| Return a flattened list of paired "create" and "upload" steps.
    : CreateMap -> { steps : StepList, upload-ids : List Text }
    = \(map : CreateMap) ->
        { steps = List/concat Step (List/map CreateEntry StepList pair-for map)
        , upload-ids = mapText upload-id (Map/keys Text CreateArgs map)
        }

in  pairs-for
