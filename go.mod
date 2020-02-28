module github.com/alexsomesan/terraform-provider-kubedynamic

go 1.12

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.3.2
	github.com/hashicorp/go-plugin v1.0.1
	github.com/hashicorp/terraform v0.12.20
	github.com/hashicorp/terraform-plugin-sdk v1.4.1
	github.com/pkg/errors v0.8.1
	github.com/zclconf/go-cty v1.2.1
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/grpc v1.23.1
	k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
)
