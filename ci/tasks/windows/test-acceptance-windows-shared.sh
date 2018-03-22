#!/bin/bash -eux
main() {
  source $PWD/bosh-dns-release/ci/assets/utils.sh

  export BBL_STATE_DIR=$PWD/bbl-state/${BBL_STATE_SUBDIRECTORY}
  source_bbl_env $BBL_STATE_DIR

  export BOSH_DEPLOYMENT=bosh-dns-shared-acceptance

  bosh -n upload-stemcell bosh-candidate-stemcell-windows/*.tgz
  bosh -n upload-stemcell gcp-linux-stemcell/*.tgz
  bosh -n upload-release candidate-release/*.tgz

  bosh -n deploy bosh-dns-release/src/bosh-dns/test_yml_assets/manifests/shared-acceptance-manifest.yml \
      --var-file bosh_ca_cert=<(echo "$BOSH_CA_CERT") \
      -v bosh_client_secret="$BOSH_CLIENT_SECRET" \
      -v bosh_client="$BOSH_CLIENT" \
      -v bosh_environment="$BOSH_ENVIRONMENT" \
      -l $BBL_STATE_DIR/vars/director-vars-store.yml \
      -l $BBL_STATE_DIR/vars/director-vars-file.yml \
      -v base_stemcell=$WINDOWS_OS_VERSION \
      -v bosh_deployment=bosh-dns \
      --vars-store dns-creds.yml

  pushd bosh-dns-release/src/bosh-dns/acceptance_tests/dns-acceptance-release
     bosh create-release --force && bosh upload-release --rebase
  popd

  bosh run-errand acceptance-tests --keep-alive
}

main
