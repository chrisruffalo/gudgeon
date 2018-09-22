package benchmarks

import (
	"runtime"
	"testing"
)

func PrintMemUsage(msg string, b *testing.B) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	b.Logf("%s: Alloc = %v MiB", msg, bToMb(m.Alloc))
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

func test(queries []string, bench Benchmark, b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := queries[i % len(queries)]
		result, err := bench.Test(q)
		if err != nil {
			b.Logf("Error during benchmark << %s >>, abort: %s", bench.Id(), err)
			return
		}
		if !result {
			b.Errorf("Did not find %s as expected", q)
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
	runtime.GC()
	PrintMemUsage("after load", b)

	// load queries to make
	queries := loadQueries()

	// reset timer and start reporting allocations
	b.ResetTimer()
	b.ReportAllocs()

	// do test(s)
	test(queries, bench, b)

	// teardown
	bench.Teardown()
}

func BenchmarkFileScan(b *testing.B) {
	bench := new(fileScan)
	benchmark(bench, b)
}

func BenchmarkWillBloom1p(b *testing.B) {
	bench := new(willbloom)
	benchmark(bench, b)
}

func BenchmarkWillBloom0_1p(b *testing.B) {
	bench := new(willbloom)
	bench.rate = 0.001
	benchmark(bench, b)
}

func BenchmarkWillBloom0_0001p(b *testing.B) {
	bench := new(willbloom)
	bench.rate = 0.000001
	benchmark(bench, b)
}

func BenchmarkWillBloom0_00000000000001p(b *testing.B) {
	bench := new(willbloom)
	bench.rate = 0.0000000000000001
	benchmark(bench, b)
}

func BenchmarkKeepFile(b *testing.B) {
	bench := new(keepfile)
	benchmark(bench, b)
}

func BenchmarkKeepFileSlow(b *testing.B) {
	bench := new(keepfileslow)
	benchmark(bench, b)
}

func BenchmarkKeepHash(b *testing.B) {
	bench := new(keephash)
	benchmark(bench, b)
}

func BenchmarkSQLStore(b *testing.B) {
	bench := new(sqlstore)
	benchmark(bench, b)
}