#!/usr/bin/env bash

cd 'cmd/coinconf'

go get -d -v ./...
go install -v ./...

package_name="coinconf-truenodes"

platforms=( "aix/ppc64" "android/arm64" "darwin/amd64" "darwin/arm64" "dragonfly/amd64" "freebsd/386" "freebsd/amd64" "freebsd/arm" "freebsd/arm64" "illumos/amd64" "linux/386" "linux/amd64" "linux/arm" "linux/arm64" "linux/mips" "linux/mips64" "linux/mips64le" "linux/mipsle" "linux/ppc64" "linux/ppc64le" "linux/riscv64" "linux/s390x" "netbsd/386" "netbsd/amd64" "netbsd/arm" "netbsd/arm64" "openbsd/386" "openbsd/amd64" "openbsd/arm" "openbsd/arm64" "openbsd/mips64" "solaris/amd64" "windows/386" "windows/amd64" "windows/arm" "windows/arm64" )
unsuported_platforms_for_now=( "android/386" "android/amd64" "android/arm" "ios/amd64" "ios/arm64" "js/wasm" "plan9/386" "plan9/amd64" "plan9/arm" )

for platform in "${platforms[@]}"
do
    echo -e "Building for\u001b[32m $platform\u001b[0m"

    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name='../../build/'$package_name'/'$package_name'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH GOARM=7 go build -a -tags netgo -ldflags '-w -s' -o $output_name .
done