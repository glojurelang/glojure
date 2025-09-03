module go.mod

go 1.25.0

// use current HEAD of the repo
require github.com/glojurelang/glojure v0.0.0

replace github.com/glojurelang/glojure => ../

require (
	bitbucket.org/pcastools/hash v1.0.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/modern-go/gls v0.0.0-20220109145502-612d0167dce5 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	go4.org/intern v0.0.0-20220617035311-6925f38cc365 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20230525183740-e7c30c78aeb2 // indirect
)
