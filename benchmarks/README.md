# Storage benchmarks

These benchmarks time the store as it is now: one map and one `sync.RWMutex`.
The results are saved in `benchmarks/baseline-single-map.txt`. After adding
sharding, the same benchmarks are run again and the results are compared.

## What is measured

- `Get` and `Set`, one at a time and from many goroutines together
- A mix of 90% reads and 10% writes
- Goroutines that all use one shared key, and goroutines that spread across
  4,096 different keys

Keys are created before the timer starts. Every value is 64 bytes, and writes
replace existing values. Only the store is measured, with no TCP or JSON.

## How to run

From the project root, in PowerShell:

```powershell
go test ./storage -run '^$' -bench '^BenchmarkStore' -benchmem '-benchtime=200ms' -count=5 '-cpu=1,4,16' |
    Tee-Object -FilePath benchmarks/baseline-single-map.txt
```

Each case runs 5 times, with 1, 4 and 16 goroutines. Don't use `-race` here,
because it slows everything down and skews the numbers.

## Comparing after sharding

After adding sharding, run the same command but save the output to
`benchmarks/sharded.txt`. Keep the baseline file so both are available.

Compare rows that have the same name, including the ending like `-4` or `-16`:

- `ns/op` is the time per operation. Lower is better. For parallel cases, it
  is the total time divided by all the operations that finished.
- `B/op` and `allocs/op` show how much memory each operation allocates.

Use the same machine and the same Go version for both runs. The baseline was run on Go 1.25.6 on Windows. 

## Checking correctness

Benchmarks only measure speed. To check that the store is still correct, run:

```powershell
go test -race ./...
```