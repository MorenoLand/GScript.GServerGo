module gserver

go 1.23

toolchain go1.24.12

require (
	github.com/MorenoLand/GScript.gs2vm-go v0.0.0
	github.com/dsnet/compress v0.0.1
	github.com/fsnotify/fsnotify v1.10.1
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20260311135729-065cd970411c // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.3.8 // indirect
)

replace github.com/MorenoLand/GScript.gs2vm-go => ../gs2vm-go
