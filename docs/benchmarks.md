# BuckTooth Benchmark Suite

## Methodology

Benchmarks live in `bench/` and use Go's built-in `testing.B` framework.
Each benchmark calls `b.ReportAllocs()` so both CPU and memory allocation
costs are visible.

- **EventBus** benchmarks measure publish latency under 1 and 10 subscribers.
- **InMemoryStore** benchmarks measure concurrent add and get throughput.
- **HTTPServer** benchmark measures constructor allocation cost.
- **Config** benchmark measures default-config YAML parse cost (no file I/O).

## Running

```bash
# All benchmarks, 5 s per benchmark, with memory stats
make bench

# Single benchmark
go test -bench=BenchmarkEventBusPublish -benchmem -benchtime=5s ./bench/...

# Run with the race detector
go test -bench=. -benchmem -race ./bench/...
```

## Baseline Results

Measured on Apple M-series, Go 1.25, single-binary build (no CGO).

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkEventBusPublish` | 204 | 136 | 3 |
| `BenchmarkEventBusPublishFanout` | 3,522 | 1,216 | 21 |
| `BenchmarkInMemoryStoreAdd` | 142 | 294 | 0 |
| `BenchmarkInMemoryStoreGet` | 365 | 1,152 | 1 |
| `BenchmarkHTTPServerNew` | 348 | 848 | 3 |
| `BenchmarkConfigLoad` | 414 | 704 | 4 |

Measured on Apple M4 Pro, Go 1.25, `GOARCH=arm64`, `-benchtime=2s`.
Run `make bench` after any significant change to capture updated numbers.

## Interpreting Results

- `ns/op` — nanoseconds per operation; lower is better.
- `B/op` — bytes allocated per operation; lower is better.
- `allocs/op` — heap allocations per operation; lower is better.

EventBus fanout scales roughly linearly with subscriber count because handlers
run concurrently but the `wg.Wait()` call serialises completion.

Config parse cost is dominated by the `gopkg.in/yaml.v3` decoder; it is only
called once at startup so the absolute value is not performance-critical.
