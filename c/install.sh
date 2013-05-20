#!/bin/sh
set -e

version=0.3.0
major_version=0.3

usage()
{
    echo "Usage: install.sh --prefix path"
}

while [ -n "${1+set}" ]; do
    case "$1" in
        -h|--help|-\?)
            usage
            exit
            ;;
        --prefix)
            if [ -z "${2+set}" ]; then
                echo "--prefix requires an argument."
                exit 1
            fi
            prefix=$2
            shift 2
            ;;
        *)
            echo "Invalid option: '$1'" 1>&2
            usage
            exit 1
            ;;
    esac
done

if [ -z "$prefix" ]; then
    echo "No prefix specified."
    usage
    exit 1
fi

case $(uname) in
    Darwin*)
        lib_file=liblucy.$version.dylib
        if [ ! -f $lib_file ]; then
            echo "$lib_file not found. Did you run make?"
            exit 1
        fi
        mkdir -p $prefix/lib
        cp $lib_file $prefix/lib
        install_name=$prefix/lib/liblucy.$major_version.dylib
        ln -sf $lib_file $install_name
        ln -sf $lib_file $prefix/lib/liblucy.dylib
        install_name_tool -id $install_name $prefix/lib/$lib_file
        ;;
    *)
        lib_file=liblucy.so.$version
        if [ ! -f $lib_file ]; then
            echo "$lib_file not found. Did you run make?"
            exit 1
        fi
        mkdir -p $prefix/lib
        cp $lib_file $prefix/lib
        soname=liblucy.so.$major_version
        ln -sf $lib_file $prefix/lib/$soname
        ln -sf $soname $prefix/lib/liblucy.so
        ;;
esac

mkdir -p $prefix/include
cp autogen/include/cfish_hostdefs.h $prefix/include
cp autogen/include/cfish_parcel.h $prefix/include
cp autogen/include/lucy_parcel.h $prefix/include
cp -R autogen/include/Clownfish $prefix/include
cp -R autogen/include/Lucy $prefix/include
cp -R autogen/include/LucyX $prefix/include

cp -R autogen/man $prefix

# create pkg-config file
# some platforms require .bak extension for temp file
cp lucy.pc.in lucy.pc
sed -i.bak "s,@version@,$version,g" lucy.pc
sed -i.bak "s,@prefix@,$prefix,g" lucy.pc
rm lucy.pc.bak
mkdir -p $prefix/lib/pkgconfig
cp lucy.pc $prefix/lib/pkgconfig
