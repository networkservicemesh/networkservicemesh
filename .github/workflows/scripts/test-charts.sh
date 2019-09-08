#!/bin/sh

CHARTS=$(ls ./deployments/helm)

lint_and_install_chart() {
    chart=$1
    echo "Installing and linting the ${chart} chart"
    docker run -v "$(pwd)":/networkservicemesh -w="/networkservicemesh/" quay.io/helmpack/chart-testing:v2.3.0 ct lint-and-install "/networkservicmsh/deployments/helm/${chart}"
}

for chart in ${CHARTS}; do
    lint_and_install_chart "${chart}"
done