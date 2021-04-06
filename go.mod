module github.com/lovethedrake/brigdrake

go 1.15

replace github.com/mholt/caddy => github.com/caddyserver/caddy/v2 v2.3.0

require (
	github.com/carolynvs/magex v0.5.0
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/google/go-github/v33 v33.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lovethedrake/drakecore v0.14.0
	github.com/magefile/mage v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.6.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	k8s.io/api v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v0.19.9
)
