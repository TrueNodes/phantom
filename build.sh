#!/usr/bin/env bash

cd 'cmd/phantom'

go get -d -v ./...
go install -v ./...

package_name="phantom-truenodes"

platforms=( "aix/ppc64" "android/386" "android/amd64" "android/arm" "android/arm64" "darwin/amd64" "darwin/arm64" "dragonfly/amd64" "freebsd/386" "freebsd/amd64" "freebsd/arm" "illumos/amd64" "ios/arm64" "js/wasm" "linux/386" "linux/amd64" "linux/arm" "linux/arm64" "linux/ppc64" "linux/ppc64le" "linux/mips" "linux/mipsle" "linux/mips64" "linux/mips64le" "linux/riscv64" "linux/s390x" "netbsd/386" "netbsd/amd64" "netbsd/arm" "openbsd/386" "openbsd/amd64" "openbsd/arm" "openbsd/arm64" "plan9/386" "plan9/amd64" "plan9/arm" "solaris/amd64" "windows/386" "windows/amd64" )

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name='../../build/'$package_name'/'$package_name'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH GOARM=7 go build -a -tags netgo -ldflags '-w' -o $output_name .
done