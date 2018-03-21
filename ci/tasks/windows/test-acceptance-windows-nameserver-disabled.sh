#!/bin/bash -eux
main() {
  source $PWD/bosh-dns-release/ci/assets/utils.sh

  export BBL_STATE_DIR=$PWD/bbl-state/${BBL_STATE_SUBDIRECTORY}
  source_bbl_env $BBL_STATE_DIR

  bosh -n upload-stemcell bosh-candidate-stemcell-windows/*.tgz
  bosh upload-release candidate-release/*.tgz

  export BOSH_DEPLOYMENT=bosh-dns-windows-acceptance

  bosh -n deploy --recreate \
      bosh-dns-release/src/bosh-dns/test_yml_assets/manifests/windows-acceptance-manifest.yml \
      -o bosh-dns-release/src/bosh-dns/acceptance_tests/windows/disable_nameserver_override/manifest-ops.yml \
      -v windows_stemcell=$WINDOWS_OS_VERSION \
      --vars-store dns-creds.yml

  bosh run-errand acceptance-tests-windows --keep-alive
}

main
