#!/bin/sh

ver=${1}
cmd=${2}
goos=${3}

[ -z $cmd ] && cmd=$(find ./cmd/ -maxdepth 1 ! -path ./cmd/ -type d 2>/dev/null)
[ -z $goos ] && goos="linux windows darwin"
[ -z $ver ] && ver=$(git describe --tags 2>/dev/null)

now=$(date --utc --iso-8601=seconds)
[ -z $ver ] && ver="untagged"

curdir=$(pwd)
rdir=$curdir/temp/release/latest
rm -rf $rdir && mkdir -p $rdir

build () {
    os=$1; arch=$2
    echo "** building for $os/$arch **"
    bdir=./temp/build/$os-$arch
    rm -rf $bdir && mkdir -p $bdir
    for c in $cmd; do
      [ ! -d $c ] && echo $c directory doesnt exist, skipping. && continue
      n=$(basename $c)
      [ "$os" != "linux" ] && [ "$n" = "dsiem" ] && continue
      [ "$os" = "windows" ] && n="${n}.exe"
      echo building $c ver=${ver} buildtime=${now} for $os: $n ..
      GOFLAGS="-mod=vendor" CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
      go build -a -ldflags "-s -w -X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" -o $bdir/$n $c || exit 1
    done
    mkdir -p $bdir/web/dist && cp -r ./web/dist/* $bdir/web/dist/ || exit 1
    cp -r ./configs ./LICENSE ./README.md $bdir/
    cd $bdir
  
    # release only the linux version of the server, there's no testing environment for Win/OSX version for this
    # and we use drwmutex that only supports Linux
    if [ "$os" = "linux" ] && [ "$arch" = "amd64" ]; then
      zname="$rdir/dsiem-server_${os}_${arch}.zip"
      echo "creating $zname .."
      dsiembin="dsiem"
      [ "$os" = "windows" ] && dsiembin="${dsiembin}.exe"
      zip -9 -r $zname $dsiembin configs web LICENSE README.md || exit 1
    fi

    tools=$(ls | grep -v dsiem | grep -v configs | grep -v web)
    zname="$rdir/dsiem-tools_${os}_${arch}.zip"
    echo "creating $zname .."
    zip -9 $zname $tools || exit 1
    cd $curdir
    rm -rf $bdir
}

echo "target OS: $goos;" commands to build: $cmd
for os in $goos; do
  build $os "amd64"
  [ "$os" = "darwin" ] && build $os "arm64"
done

cd $rdir
for f in $(ls $rdir | grep -v sha256); do
  sha256sum $f > $f.sha256.txt || exit 1
done
echo "Done building, content of $rdir:"
ls -l $rdir
