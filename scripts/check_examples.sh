#!/usr/bin/env bash

set -e

export TF_IN_AUTOMATION=true

if [ ! -f ./terraform-provider-kubernetes-alpha ]; then
    make build
fi

FILE=.check_validate_only
for example in $PWD/examples/*; do
    cd $example
    echo 🔍 $(tput bold)$(tput setaf 3)Checking $(basename $example)...
        
    if [ -f "$FILE" ]; then
        echo "$FILE exists. Only running `terraform validate`"
        terraform init -plugin-dir ../..
        terraform validate
        echo
    else 
        terraform init -plugin-dir ../..
        terraform validate
        terraform plan -out tfplan > /dev/null
        terraform apply tfplan
        terraform refresh
        terraform destroy -auto-approve
        echo
    fi
    
done