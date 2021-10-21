module gerrit.oran-osc.org/r/ric-plt/o1mediator

go 1.12

replace gerrit.o-ran-sc.org/r/ric-plt/xapp-frame => gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git v0.9.1

replace gerrit.o-ran-sc.org/r/ric-plt/sdlgo => gerrit.o-ran-sc.org/r/ric-plt/sdlgo.git v0.7.0

replace gerrit.o-ran-sc.org/r/com/golog => gerrit.o-ran-sc.org/r/com/golog.git v0.0.2

require (
	gerrit.o-ran-sc.org/r/ric-plt/xapp-frame v0.0.0-00010101000000-000000000000
	github.com/Juniper/go-netconf v0.1.1
	github.com/basgys/goxml2json v1.1.0
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/go-openapi/errors v0.19.3
	github.com/go-openapi/runtime v0.19.7
	github.com/go-openapi/spec v0.19.4 // indirect
	github.com/go-openapi/strfmt v0.19.4
	github.com/go-openapi/swag v0.19.7
	github.com/go-openapi/validate v0.19.6
	github.com/prometheus/alertmanager v0.20.0
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/valyala/fastjson v1.4.1
	github.com/ziutek/telnet v0.0.0-20180329124119-c3b780dc415b // indirect
	golang.org/x/crypto v0.0.0-20190617133340-57b3e21c3d56
)
