
<a name="v1.6.0"></a>
## [v1.6.0](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.4.0...v1.6.0) (2024-04-21)

Welcome to the v1.6.0 release of Talos CCM!

### Features
- support CloudDualStackNodeIPs
- deploy without cni
- sign images

### Changelog

* 27aa781 chore: bump deps
* 9d65a90 chore: bump deps
* 9403bc5 fix: refresh talos tls certs
* b4e136b feat: support CloudDualStackNodeIPs
* 670ead7 feat: deploy without cni
* 33faa60 chore: bump deps
* 3c9d805 fix: prepend v for image
* 5d41626 fix: azure providerID
* eff652f chore: bump deps
* 214cc87 chore: bump deps
* 5a1eaf7 chore: bump deps
* fe5a0b1 chore: bump github actions deps
* 562e738 feat: sign images

<a name="v1.4.0"></a>
## [v1.4.0](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.3.0...v1.4.0) (2023-05-27)

Welcome to the v1.4.0 release of Talos CCM!

### Features
- add node certificate approval
- build latest version
- daemonset deployment
- label spot instanses

### Changelog

* b3d55f0 test: add basic tests
* e44f5bc chore: bump deps
* 3dcea64 docs: edge deploy with csr
* bba5b6a docs: update helm readme
* 5d65b1d fix: csr keyusage check
* 2b53c2b feat: add node certificate approval
* 11e77e8 feat: build latest version
* 7a039d9 fix: node spec ip
* 8583f59 chore: bump deps
* 8681816 feat: daemonset deployment
* 5a4413f chore: bump deps
* c80d552 feat: label spot instanses
* 9e1b15e chore: bump deps
* d3d613b fix: helm chart namespace

<a name="v1.3.0"></a>
## v1.3.0 (2022-12-20)

Welcome to the v1.3.0 release of Talos CCM!

### Features
- gitops automatization
- init ccm

### Changelog

* e8a9802 feat: gitops automatization
* 70777c7 docs: update readme
* e34ca47 chore: update go.mod
* 9825766 fix: helm chart tolerations
* 345c59f feat: init ccm
