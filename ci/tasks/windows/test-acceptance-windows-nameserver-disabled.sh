#!/bin/bash -eux
main() {
  source $PWD/bosh-dns-release/ci/assets/utils.sh

  export BBL_STATE_DIR=$PWD/bbl-state/${BBL_STATE_SUBDIRECTORY}
  source_bbl_env $BBL_STATE_DIR

  export BOSH_DEPLOYMENT=bosh-dns-windows-nameserver-disable-acceptance

  bosh -n upload-stemcell bosh-candidate-stemcell-windows/*.tgz
  bosh -n upload-release candidate-release/*.tgz

  bosh -n deploy \
      bosh-dns-release/src/bosh-dns/test_yml_assets/manifests/windows-acceptance-manifest.yml \
      -o bosh-dns-release/src/bosh-dns/acceptance_tests/windows/disable_nameserver_override/manifest-ops.yml \
      -v windows_stemcell=$WINDOWS_OS_VERSION \
      -v deployment_name=bosh-dns-windows-acceptance \
      --vars-store dns-creds.yml

  bosh run-errand acceptance-tests-windows --keep-alive
}

main
