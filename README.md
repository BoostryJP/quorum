<p align="center">
  <img width="33%" src="https://user-images.githubusercontent.com/963333/212965270-bba794c6-be66-4850-9475-19895530f32c.png"/>
</p>

# ibet-Core

<p>
  <img alt="Version" src="https://img.shields.io/badge/version-26.9-blue.svg?cacheSeconds=2592000" />
</p>

ibet-Core is the node client for [ibet Network](https://github.com/BoostryJP/ibet-Network), developed from a [GoQuorum](https://github.com/ConsenSys/quorum) fork and maintained independently by BOOSTRY.

## Overview

ibet-Core started as a fork of GoQuorum, which itself is based on go-ethereum. Since the fork, it has been developed independently for ibet Network rather than tracking new GoQuorum releases.
It remains an EVM-compatible blockchain node client and selectively incorporates go-ethereum execution-layer improvements, dependency updates, and ibet Network-specific features needed for efficient and secure operation.

The version control policy of this project follows that of ibet-Network.

## Features

ibet-Core is developed on the GoQuorum baseline adopted by this project and includes the following enhancements:
- The default block generation interval is set to 1 second.
- Go 1.26 is supported, with third-party packages updated from a security perspective.
- Selected go-ethereum execution-layer improvements are incorporated where they benefit ibet Network.
- Project-specific fixes and operational improvements are maintained independently after the GoQuorum fork.
- Added precompile for secp256r1 signature verification ([EIP-7951](https://eips.ethereum.org/EIPS/eip-7951)).
- Added precompile for BLS12-381 curve operations ([EIP-2537](https://eips.ethereum.org/EIPS/eip-2537)).
- Added support for AWS Secrets Manager / KMS based node key management.

## Building ibet-Core
Building ibet-Core requires both a Go (version 1.26) and a C compiler.
You can install them using your favourite package manager. 
Once the dependencies are installed, run
```
make geth
```

or, to build the full suite of utilities:
```
make all
```

## AWS node key management

For AWS Secrets Manager / KMS based node key setup, see
[docs/aws-nodekey.md](docs/aws-nodekey.md).

## License

ibet-Core is derived from GoQuorum and go-ethereum. This name change does not change the licenses of the upstream-derived code or bundled third-party components.

The go-ethereum library code (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in this repository in the `COPYING.LESSER` file.

The go-ethereum binary code (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in this repository in the `COPYING` file.

Any project planning to use the `crypto/secp256k1` sub-module must use the specific [secp256k1 standalone library](https://github.com/ConsenSys/goquorum-crypto-secp256k1) licensed under 3-clause BSD.
