Collatz sequence solver written in Go

Collatz Conjecture
- start with any positive integer n
- if n is even, the next term is n/2
- if n is odd, the next term is 3n + 1
- no matter the value of n, the sequence always reaches 1

This program computes a range of n, printing records of how long the chain of steps was to compute n down to 1. It also features some basic functionality to track previously computed chain values in a hashmap and check current calculation values against the map to speed up pre-computed values. The 'profiler.go' file aims to keep that hashmap size in check to not eat all the memory.

This is written in pure Go, but not as fast as my collatz-cgo implementation.
