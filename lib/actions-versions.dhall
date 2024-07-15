let cache = "v4"

in  [ "actions/cache@${cache}"
    , "actions/cache/restore@${cache}"
    , "actions/cache/save@${cache}"
    , "actions/checkout@v4"
    , "actions/github-script@v7"
    , "actions/setup-go@v5"
    , "actions/upload-artifact@v4"
    ]
