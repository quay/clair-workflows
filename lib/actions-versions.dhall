let cache = "v5"

in  [ "actions/cache@${cache}"
    , "actions/cache/restore@${cache}"
    , "actions/cache/save@${cache}"
    , "actions/checkout@v5"
    , "actions/github-script@v8"
    , "actions/setup-go@v6"
    , "actions/upload-artifact@v6"
    , "codecov/codecov-action@v5"
    ]
