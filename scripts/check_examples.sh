#!/usr/bin/env bash

set -e

export TF_IN_AUTOMATION=true

if [ ! -f ./terraform-provider-kubernetes-alpha ]; then
    make build
fi

for example in $PWD/examples/*; do
    cd $example
    echo 🔍 $(tput bold)$(tput setaf 3)Checking $(basename $example)...
    terraform init -plugin-dir ../..
    terraform validate
    terraform plan -out tfplan > /dev/null
    terraform apply tfplan
    terraform refresh
    terraform destroy -auto-approve
    echo
done