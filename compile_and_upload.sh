#!/usr/bin/env bash

cd 'cmd/phantom'

go get -d -v ./...
go install -v ./...

package_name=phantom

platforms=("windows/amd64" "linux/amd64" "darwin/amd64" "linux/arm" )

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o TrueNodes-phantom *.go

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name='../../'$package_name'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    env GOOS=$GOOS GOARCH=$GOARCH GOARM=7 go build -o $output_name .
    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
done

scp -r TrueNodes-phantom localparacopiar