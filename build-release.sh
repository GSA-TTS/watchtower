#!/bin/bash -e

if [ -z "$1" ]; then 
    echo "Usage: ./build-release <version-number>"
    exit
fi
version="$1"

package="github.com/18f/watchtower"
package_split=(${package//\// })
package_name=${package_split[-1]}

platforms=("darwin/amd64" "darwin/arm64" "windows/amd64" "windows/386" "linux/386" "linux/amd64" "linux/arm" "linux/arm64" "linux/ppc64" "linux/ppc64le" "linux/mips" "linux/mipsle" "linux/mips64" "linux/mips64le" "netbsd/386" "netbsd/amd64" "netbsd/arm" "openbsd/386" "openbsd/amd64" "openbsd/arm" "freebsd/386" "freebsd/amd64" "freebsd/arm" "plan9/386" "plan9/amd64" "solaris/amd64")

build_dir="build_output"
rm -rf $build_dir
mkdir $build_dir
cd $build_dir

touch sha256sums.txt
windows_suffix=".exe"

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name=$package_name'-'$version'.'$GOOS'-'$GOARCH
    if [ "$GOOS" = "windows" ]; then
        output_name+=$windows_suffix
    fi

    if ! env GOOS="$GOOS" GOARCH="$GOARCH" go build -o $output_name $package ; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi

    # compressed = output_name - ".exe" suffix if it exists
    compressed_name=${output_name%"$windows_suffix"}

    tar -czvf $compressed_name.tar.gz $output_name
    if [ "$GOOS" = "windows" ]; then
        zip -r $compressed_name.zip $output_name
    fi

    sha256sum $compressed_name.tar.gz >> sha256sums.txt
    if [ "$GOOS" = "windows" ]; then
        sha256sum $compressed_name.zip >> sha256sums.txt
    fi

    rm $output_name
    echo "Finished building $GOOS / $GOARCH"
done
