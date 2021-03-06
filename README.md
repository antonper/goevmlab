# Go evmlab

This project is inspired by [EVMlab](https://github.com/ethereum/evmlab), which was
written in Python. EVMlab featured a minimal "compiler", along with some tooling
to view traces in a UI, and execute scripts against EVMs (parity and geth). 

This is a golang rewrite from scratch of that same project, this time in go-lang to be more stable 
and nice to use. 

## Status

So far, it only contains 

- a minimal "compiler", which is _not_ a proper compiler, but it can be used to create EVM bytecode and do some basic higher level operations. 
- Some infrastructure for writing EVM-based fuzzers

## Examples

See [examples/calltree](the calltree example) to get an idea of how to use this
thing, along with an [analysis](examples/calltree/README.md) done using 
this framework. 