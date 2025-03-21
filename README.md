<p align="center">
  <img width="33%" src="https://user-images.githubusercontent.com/963333/212965270-bba794c6-be66-4850-9475-19895530f32c.png"/>
</p>

# GoQuorum for ibet Network

<p>
  <img alt="Version" src="https://img.shields.io/badge/version-2.5-blue.svg?cacheSeconds=2592000" />
</p>

This project is [GoQuorum](https://github.com/ConsenSys/quorum) fork for [ibet Network](https://github.com/BoostryJP/ibet-Network)

## Version control policy

This project is a fork of GoQuorum (and go-ethereum), and we decide which version of GoQuorum to adopt when developing each version and reflect those modules. 
In addition, it has been modified to be optimal for the ibet Network.

The version control policy of this project follows that of ibet-Network.

## Reference GoQuorum version

Currently, the ibet Network is built using a node client based on v24.4.0 of GoQuorum. 
However, it has been variously patched to be optimized for ibet Network. For example:
- The default block generation interval is set to 1 second.
- Fully supports Go 1.23 and applies new 3rd party packages from a security perspective.
- Made temporary fixes for bugs before they were fixed in the original GoQuorum.

## Building the source
Building quorum requires both a Go (version 1.23) and a C compiler. 
You can install them using your favourite package manager. 
Once the dependencies are installed, run
```
make geth
```

or, to build the full suite of utilities:
```
make all
```

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.

Any project planning to use the `crypto/secp256k1` sub-module must use the specific [secp256k1 standalone library](https://github.com/ConsenSys/goquorum-crypto-secp256k1) licensed under 3-clause BSD.

