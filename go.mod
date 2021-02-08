module github.com/hashicorp/terraform-provider-kubernetes-alpha

go 1.15

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/color v1.9.0 // indirect
	github.com/getkin/kin-openapi v0.22.1
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hashicorp/terraform-exec v0.10.0
	github.com/hashicorp/terraform-json v0.5.0
	github.com/hashicorp/terraform-plugin-go v0.2.1
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.2.0
	github.com/hashicorp/terraform-plugin-test/v2 v2.1.2
	github.com/hashicorp/yamux v0.0.0-20200609203250-aecfd211c9ce // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/hashstructure v1.0.0
	github.com/oklog/run v1.1.0 // indirect
	github.com/stretchr/testify v1.6.1
	google.golang.org/genproto v0.0.0-20201030142918-24207fddd1c3 // indirect
	google.golang.org/grpc v1.33.1
	k8s.io/apiextensions-apiserver v0.18.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
)

replace github.com/hashicorp/terraform-plugin-go => /home/alex/work/terraform-plugin-go // TODO: Remove once https://github.com/hashicorp/terraform-plugin-go/pull/58 is merged
