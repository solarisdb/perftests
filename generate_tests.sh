#!/bin/bash

# generate cleanup
./build/perftests generateCfg auto cleanup
#------------------
# generate sleep
./build/perftests generateCfg auto sleep
#---- 20 logs by 1GB, 1 writer, batch 500x --------------
# generate append to 20 log 1GB size by 1 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 20 1073741824 1 500 102400

# generate append to 20 log 1GB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 20 1073741824 1 500 10240

# generate append to 20 log 1GB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 20 1073741824 1 500 1024
#------------------
# generate append to 1 log 1GB size by 1 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 1 1073741824 1 500 102400

# generate append to 1 log 1GB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 1 1073741824 1 500 10240

# generate append to 1 log 1GB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 1 1073741824 1 500 1024
#------------------
# generate append to 1 log 1GB size by 10 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 1 1073741824 10 500 102400

# generate append to 1 log 1GB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 1 1073741824 10 500 10240

# generate append to 1 log 1GB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 1 1073741824 10 500 1024
#------------------
# generate append to 10 log 100MB size by 1 writer, batch 500 by 100KB
./build/perftests generateCfg auto append 10 107374182 1 500 102400

# generate append to 10 log 100MB size by 1 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 10 107374182 1 500 10240

# generate append to 10 log 100MB size by 1 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 10 107374182 1 500 1024
#------------------
# generate append to 1 log 1GB size by 1K writer, batch 500 by 100KB
./build/perftests generateCfg auto append 1 1073741824 1000 500 102400

# generate append to 1 log 1GB size by 1000 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 1 1073741824 1000 500 10240

# generate append to 1 log 1GB size by 1000 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 1 1073741824 1000 500 1024
#------------------
# generate append to 10 log 100MB size by 1K writer, batch 500 by 100KB
./build/perftests generateCfg auto append 10 107374182 1000 500 102400

# generate append to 10 log 100MB  size by 1000 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 10 107374182 1000 500 10240

# generate append to 10 log 100MB  size by 1000 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 10 107374182 1000 500 1024
#------------------
# generate append to 100 log 10MB size by 1K writer, batch 500 by 100KB
./build/perftests generateCfg auto append 100 10737418 1000 500 102400

# generate append to 100 log 10MB  size by 1000 writer, batch 500 by 10KB
./build/perftests generateCfg auto append 100 10737418 1000 500 10240

# generate append to 100 log 10MB  size by 1000 writer, batch 500 by 1KB
./build/perftests generateCfg auto append 100 10737418 1000 500 1024
#------------------
# generate append to 100 log 1MB size by 100 writer, batch 1 by 100B
./build/perftests generateCfg auto append 100 1024000 100 1 100
#------------------
# generate append to 200 log 1MB size by 100 writer, batch 1 by 100B
./build/perftests generateCfg auto append 200 1024000 100 1 100
#------------------

#-------seq query--------
# 1GB=1073741824
# generate seq query from 1 logs 2GB size by 1 readers, batch 100 by 1KB
./build/perftests generateCfg auto seq_query 1 2143741824 1 100 1024
#------------------
# generate seq query from 1 logs 2GB size by 1 readers, batch 100 by 10KB
./build/perftests generateCfg auto seq_query 1 2143741824 1 100 10240
#------------------
# generate seq query from 1 logs 2GB size by 1 readers, batch 100 by 100KB
./build/perftests generateCfg auto seq_query 1 2143741824 1 100 102400
#------------------
# generate seq query from 10 logs 2GB size by 10 readers, batch 500 by 1KB
./build/perftests generateCfg auto seq_query 10 2143741824 10 500 1024
#------------------
# generate seq query from 10 logs 2GB size by 10 readers, batch 500 by 10KB
./build/perftests generateCfg auto seq_query 10 2143741824 10 500 10240
#------------------
# generate seq query from 10 logs 2GB size by 10 readers, batch 500 by 100KB
./build/perftests generateCfg auto seq_query 10 2143741824 10 500 102400
#------------------

#-------rand query--------
# 1GB=1073741824
# generate seq query from 1 logs 2GB size by 1 readers, batch 100 by 1KB
./build/perftests generateCfg auto rand_query 1 2143741824 1 100 1024
#------------------
# generate seq query from 1 logs 2GB size by 1 readers, batch 100 by 10KB
./build/perftests generateCfg auto rand_query 1 2143741824 1 100 10240
#------------------
# generate seq query from 1 logs 2GB size by 1 readers, batch 100 by 100KB
./build/perftests generateCfg auto rand_query 1 2143741824 1 100 102400
#------------------
# generate seq query from 10 logs 2GB size by 10 readers, batch 500 by 1KB
./build/perftests generateCfg auto rand_query 10 2143741824 10 500 1024
#------------------
# generate seq query from 10 logs 2GB size by 10 readers, batch 500 by 10KB
./build/perftests generateCfg auto rand_query 10 2143741824 10 500 10240
#------------------
# generate seq query from 10 logs 2GB size by 10 readers, batch 500 by 100KB
./build/perftests generateCfg auto rand_query 10 2143741824 10 500 102400
#------------------