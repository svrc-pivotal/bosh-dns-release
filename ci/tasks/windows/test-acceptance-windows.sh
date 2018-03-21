#!/bin/bash -eux
main() {
  source $PWD/bosh-dns-release/ci/assets/utils.sh

  export BBL_STATE_DIR=$PWD/bbl-state/${BBL_STATE_SUBDIRECTORY}
  source_bbl_env $BBL_STATE_DIR

  export BOSH_DEPLOYMENT=bosh-dns-windows-acceptance

  bosh -n upload-stemcell bosh-candidate-stemcell-windows/*.tgz
  bosh -n upload-release candidate-release/*.tgz

  bosh -n deploy bosh-dns-release/src/bosh-dns/test_yml_assets/manifests/windows-acceptance-manifest.yml \
    -v health_server_port=2345 \
    -v windows_stemcell=$WINDOWS_OS_VERSION \
    -o bosh-dns-release/src/bosh-dns/test_yml_assets/ops/enable-health-manifest-ops.yml \
    --vars-store dns-creds.yml

  bosh run-errand acceptance-tests-windows
}
