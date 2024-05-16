#!/bin/bash

# generate cleanup
./build/perftests generateCfg auto cleanup
#------------------
# generate sleep
./build/perftests generateCfg auto sleep
#------------------
# generate append to 1 log 2GB size by 1 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 1 2147483648 1 500 102400

# generate append to 1 log 2GB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 1 2147483648 1 500 10240

# generate append to 1 log 2GB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 1 2147483648 1 500 1024
#------------------
# generate append to 1 log 2GB size by 10 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 1 2147483648 10 500 102400

# generate append to 1 log 2GB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 1 2147483648 10 500 10240

# generate append to 1 log 2GB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 1 2147483648 10 500 1024
#------------------
# generate append to 10 log 200MB size by 1 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 10 214748364 1 500 102400

# generate append to 10 log 200MB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 10 214748364 1 500 10240

# generate append to 10 log 200MB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 10 214748364 1 500 1024
#------------------
# generate append to 1 log 2GB size by 1K writer, batch 500 by 100KB
./build/perftests generateCfg auto append 1 2147483648 1000 500 102400

# generate append to 1 log 2GB size by 1000 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 1 2147483648 1000 500 10240

# generate append to 1 log 2GB size by 1000 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 1 2147483648 1000 500 1024
#------------------
# generate append to 10 log 200MB size by 1K writer, batch 500 by 100KB
./build/perftests generateCfg auto append 10 214748364 1000 500 102400

# generate append to 10 log 200MB  size by 1000 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 10 214748364 1000 500 10240

# generate append to 10 log 200MB  size by 1000 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 10 214748364 1000 500 1024
#------------------
# generate append to 100 log 20MB size by 1K writer, batch 500 by 100KB
./build/perftests generateCfg auto append 100 21474836 1000 500 102400

# generate append to 100 log 20MB  size by 1000 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 100 21474836 1000 500 10240

# generate append to 100 log 20MB  size by 1000 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 100 21474836 1000 500 1024
#------------------
# generate append to 100K log 100KB size by 100 writer, batch 1 by 100B
./build/perftests generateCfg auto append 100000 102400 100 1 100
#------------------