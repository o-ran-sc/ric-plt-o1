module gerrit.oran-osc.org/r/ric-plt/o1mediator

go 1.12

replace gerrit.o-ran-sc.org/r/ric-plt/xapp-frame => gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git v0.8.1

replace gerrit.o-ran-sc.org/r/ric-plt/sdlgo => gerrit.o-ran-sc.org/r/ric-plt/sdlgo.git v0.5.0

replace gerrit.o-ran-sc.org/r/com/golog => gerrit.o-ran-sc.org/r/com/golog.git v0.0.2

require (
	gerrit.o-ran-sc.org/r/com/golog v0.0.2
	gerrit.o-ran-sc.org/r/ric-plt/xapp-frame v0.0.0-00010101000000-000000000000
	github.com/Juniper/go-netconf v0.1.1 // indirect
	github.com/basgys/goxml2json v1.1.0 // indirect
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-openapi/errors v0.19.3
	github.com/go-openapi/loads v0.19.4
	github.com/go-openapi/runtime v0.19.7
	github.com/go-openapi/spec v0.19.4
	github.com/go-openapi/strfmt v0.19.4
	github.com/go-openapi/swag v0.19.7
	github.com/go-openapi/validate v0.19.6
	github.com/gorilla/mux v1.7.1
	github.com/jessevdk/go-flags v1.4.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/orcaman/concurrent-map v0.0.0-20190314100340-2693aad1ed75
	github.com/prometheus/alertmanager v0.20.0
	github.com/segmentio/ksuid v1.0.2
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	github.com/valyala/fastjson v1.4.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.1.0
	github.com/ziutek/telnet v0.0.0-20180329124119-c3b780dc415b // indirect
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297
)
