module github.com/defenxor/dsiem

require (
	github.com/allegro/bigcache v1.2.1
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dogenzaka/tsv v0.0.0-20150215104501-8e02e611b1fb
	github.com/enriquebris/goconcurrentqueue v0.0.0-20190719205347-3e5689c24f05
	github.com/fasthttp-contrib/websocket v0.0.0-20160511215533-1f3b11f56072
	github.com/gocarina/gocsv/v2 v2.0.0-20181026075406-cde31a6ec2a8
	github.com/jonhoo/drwmutex v0.0.0-20190519183033-0cffe0733098
	github.com/julienschmidt/httprouter v1.3.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/nats-io/nats-server/v2 v2.7.4
	github.com/nats-io/nats.go v1.13.1-0.20220308171302-2f2f6968e98d
	github.com/olivere/elastic v6.2.25+incompatible
	github.com/olivere/elastic/v7 v7.0.8
	github.com/paulbellamy/ratecounter v0.2.0
	github.com/pkg/profile v1.3.0
	github.com/remeh/sizedwaitgroup v0.0.0-20180822144253-5e7302b12cce
	github.com/satori/go.uuid v0.0.0-20180103174451-36e9d2ebbde5
	github.com/sebdah/goldie v0.0.0-20180424091453-8784dd1ab561
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	github.com/valyala/fasthttp v1.34.0
	github.com/valyala/tsvreader v1.0.0
	github.com/yl2chen/cidranger v0.0.0-20190806234802-fed7223fd934
	go.elastic.co/apm v1.11.0
	go.elastic.co/apm/module/apmhttp v1.11.0
	go.uber.org/zap v1.10.0
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11
	gopkg.in/olivere/elastic.v5 v5.0.82
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/elastic/go-sysinfo v1.1.1 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/nats-io/jwt/v2 v2.2.1-0.20220113022732-58e87895b296 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml v1.5.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.2.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

go 1.19
