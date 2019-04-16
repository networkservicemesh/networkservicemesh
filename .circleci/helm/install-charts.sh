#!/bin/sh

CHARTS=$(ls ./deployments/helm)

for chart in ${CHARTS}; do
    ct lint-and-install "./deployments/helm/${chart}"
done
