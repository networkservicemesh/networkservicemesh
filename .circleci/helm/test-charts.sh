#!/bin/sh

CHARTS=$(ls ./deployments/helm)

lint_and_install_chart() {
    chart=$1
    echo "Installing and linting the ${chart} chart"
    ct lint-and-install "./deployments/helm/${chart}"
}

for chart in ${CHARTS}; do
    lint_and_install_chart "${chart}"
done
