#! /bin/sh

git clone https://github.com/libgit2/libgit2.git ~/libgit2
cd ~/libgit2
git checkout v1.1.0
mkdir build && cd build
cmake .. -DBUILD_CLAR=0 -DBUILD_SHARED_LIBS=0
cmake --build . --target install
