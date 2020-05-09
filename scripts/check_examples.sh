#!/usr/bin/bash
set -e

export TF_IN_AUTOMATION=true

if [ ! -f ./terraform-provider-kubernetes-alpha ]; then
    make build
fi

for example in $PWD/examples/*; do
    cd $example
    terraform init -plugin-dir ../..
    terraform validate
    terraform plan -out tfplan > /dev/null
    terraform apply tfplan
    terraform refresh
    terraform destroy -auto-approve
done