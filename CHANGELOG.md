
<a name="v1.11.0"></a>
## [v1.11.0](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.10.1...v1.11.0) (2025-09-17)

Welcome to the v1.11.0 release of Talos CCM!

### Changelog

* 7509491 fix: csr dns name check
* 4b4c758 fix: service account name
* 4402b31 chore: bump deps
* 9c000cf chore: bump deps

<a name="v1.10.1"></a>
## [v1.10.1](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.10.0...v1.10.1) (2025-06-27)

Welcome to the v1.10.1 release of Talos CCM!

### Changelog

* bbe9294 chore: bump deps

<a name="v1.10.0"></a>
## [v1.10.0](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.9.1...v1.10.0) (2025-06-05)

Welcome to the v1.10.0 release of Talos CCM!

### Changelog

* 01d526d chore: bump deps
* 95b4c4b fix: ipv6 small subnets
* a0e8169 chore: bump deps

<a name="v1.9.1"></a>
## [v1.9.1](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.9.0...v1.9.1) (2025-04-11)

Welcome to the v1.9.1 release of Talos CCM!

### Features
- add special transformer func
- add system information for transformer
- renovate bot
- add taints capabilities

### Changelog

* 470f45c chore: bump deps
* 094360a fix: hostname in transformation rules
* dc5bfc4 chore: bump deps
* 2c0bd2f feat: add special transformer func
* 5a31bb2 feat: add system information for transformer
* 67f83c6 feat: renovate bot
* 82c154a feat: add taints capabilities

<a name="v1.9.0"></a>
## [v1.9.0](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.8.1...v1.9.0) (2025-01-03)

Welcome to the v1.9.0 release of Talos CCM!

### Changelog

* adb835e chore: bump deps
* 2cfa7c6 chore: bump deps

<a name="v1.8.1"></a>
## [v1.8.1](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.8.0...v1.8.1) (2024-10-21)

Welcome to the v1.8.1 release of Talos CCM!

### Features
- ipv6 small subnets
- make kube-apiserver endpoint configurable

### Changelog

* 82009ed feat: ipv6 small subnets
* 68d4133 fix: node allocator
* db6c211 docs: install troubleshoot
* 628a7b7 feat: make kube-apiserver endpoint configurable

<a name="v1.8.0"></a>
## [v1.8.0](https://github.com/siderolabs/talos-cloud-controller-manager/compare/v1.6.0...v1.8.0) (2024-09-24)

Welcome to the v1.8.0 release of Talos CCM!

### Features
- gcp spot instances
- node ipam controller
- prefer permanent ipv6
- transformer functions
- expose metrics
- node transformer feature flags
- node transformer

### Changelog

* 8350f49 chore: bump deps
* 01145da docs: update deploy documentation
* 09a5b9e refactor: csr approval controller
* 31c9b5b docs: split readme file
* 122019a chore: bump deps
* 326fc53 feat: gcp spot instances
* e1a0e0e feat: node ipam controller
* 3b20bb0 refactor: contextual logging
* 3a4ae03 feat: prefer permanent ipv6
* 7dac5b8 fix: set priorityClassName
* 53034c8 chore: clean flag
* 9dde8aa fix: empty terms
* 749a01d fix: make possible mutate provider-id
* c0988a3 docs: add config documentation
* 386958d feat: transformer functions
* 0e8728c feat: expose metrics
* 0faf0ae fix: refresh talos token
* 85e2022 feat: node transformer feature flags
* 22e3984 feat: node transformer

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
