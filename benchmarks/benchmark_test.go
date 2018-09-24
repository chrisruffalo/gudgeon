package benchmarks

import (
	"runtime"
	"testing"
)

type query struct {
	domain   string
	found bool
}

var queries = []query{
	// near the top of the list
	{ domain: "google.com", found: true },
	{ domain: "amazon.com", found: true },
	{ domain: "netflix.com", found: true },

	// middle of list
	{ domain: "www.missmoss.co.za", found: true },
	{ domain: "www.mmsend30.com", found: true },
	{ domain: "www.monat.mx", found: true },

	// very bottom of list
	{ domain: "www.price4limo.com", found: true },
	{ domain: "www.probuilder.com", found: true },
	{ domain: "www.professorshouse.com", found: true },

	// subdomains of listed domains
	{ domain: "ads.google.com", found: true },
	{ domain: "subnet.netflix.com", found: true },
	{ domain: "things.www.monat.mx", found: true },
	{ domain: "test.www.mmsend30.com", found: true },
	{ domain: "thisisnotasubdomain.google.com", found: true },
	{ domain: "nowaythisisfoundasadomain.www.professorshouse.com", found: true },

	// domains not in list
	{ domain: "gudgeon.com", found: false },
	{ domain: "gyip.io", found: false },
	{ domain: "testdomainthatisntfound.com", found: false },
	{ domain: "homeagainhomeagain.com", found: false },
	{ domain: "w.com", found: false },
}

func PrintMemUsage(msg string, b *testing.B) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	b.Logf("%s: Alloc = %v MiB", msg, bToMb(m.Alloc))
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}

func test(bench Benchmark, b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := queries[i % len(queries)]
		result, err := bench.Test(q.domain)
		if err != nil {
			b.Logf("Error during benchmark << %s >>, abort: %s", bench.Id(), err)
			return
		}
		if result != q.found {
			b.Errorf("Result %t for %s but expected %t", result, q.domain, q.found)
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

	// reset timer and start reporting allocations
	b.ResetTimer()
	b.ReportAllocs()

	// do test(s)
	test(bench, b)

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