module github.com/chrisruffalo/gudgeon

require (
	github.com/GeertJohan/go.rice v1.0.1-0.20191102153406-d954009f7238
	github.com/akutz/sortfold v0.2.1
	github.com/atrox/go-migrate-rice v1.0.1
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/couchbase/go-slab v0.0.0-20150629231827-1f5f7f282713
	github.com/fortytw2/leaktest v1.3.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gin-gonic/gin v1.6.2
	github.com/golang-migrate/migrate/v4 v4.10.0
	github.com/google/uuid v1.0.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/json-iterator/go v1.1.9
	github.com/miekg/dns v1.1.29
	github.com/mina86/unsafeConvert v0.0.0-20170228191759-4dde7f529f51
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/ryanuber/go-glob v1.0.0
	github.com/shirou/gopsutil v0.0.0-20190627142359-4c8b404ee5c5
	github.com/sirupsen/logrus v1.5.0
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 // indirect
	github.com/twmb/murmur3 v1.1.3
	github.com/willf/bitset v1.1.10 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v2.0.3+incompatible

go 1.14
