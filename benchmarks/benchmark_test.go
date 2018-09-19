package benchmarks

import (
	"runtime"
	"testing"
)

func PrintMemUsage(msg string, b *testing.B) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    // For info on each, see: https://golang.org/pkg/runtime/#MemStats
    b.Logf("%s: Alloc = %v MiB, TotalAlloc = %v MiB", msg, bToMb(m.Alloc), bToMb(m.TotalAlloc))
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}

func loadQueries() []string {
	queries := []string{
		// in the list or near the top of it
		"google.com",
		"amazon.com",
		"netflix.com",

		// near the middle
		"www.missmoss.co.za",
		"www.mmsend30.com",
		"www.monat.mx",

		// at the bottom
		"www.price4limo.com",
		"www.probuilder.com",
		"www.professorshouse.com",

		// dns children for suffix match rule
		"ads.google.com",
		"subnet.netflix.com",
		"things.www.monat.mx",
		"test.www.mmsend30.com",
		"thisisnotasubdomain.google.com",
		"nowaythisisfoundasadomain.www.professorshouse.com",
	} 

	return queries
}

func test(queries []string, bench Benchmark, b *testing.B, pb *testing.PB) {
	for pb.Next() {
		found := 0
		for _, q := range queries {
			result, err := bench.Test(q)
			if err != nil {
				b.Errorf("Error during benchmark, abort: %s", err)
				return
			}
			if result {
				found ++
			}
		}
		if found != len(queries) {
			b.Errorf("Found %d but expected %d", found, len(queries))
		}
	}
}

func benchmark(bench Benchmark, b *testing.B) {
	// do pre-load stuff
	PrintMemUsage("before load", b)
	err := bench.Load("testdata/top-1m.list")
	if err != nil {
		b.Errorf("Could not load benchmark: %s", err)
		return
	}
	PrintMemUsage("after load", b)

	// load queries to make
	queries := loadQueries()

	// reset timer and start reporting allocations
	b.ResetTimer()
	b.ReportAllocs()

	// do test(s)
	b.RunParallel(func(pb *testing.PB){
		test(queries, bench, b, pb)
	})

	PrintMemUsage("after test", b)
}

// implement filescan benchmark
func BenchmarkFileScan(b *testing.B) {
	bench := new(fileScan)
	benchmark(bench, b)
}

func BenchmarkMPH(b *testing.B) {
	bench := new(minhash)
	benchmark(bench, b)
}

func BenchmarkWillBloom(b *testing.B) {
	bench := new(willbloom)
	benchmark(bench, b)
}