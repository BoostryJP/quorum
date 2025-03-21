// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/http"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/extension/privacyExtension"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/permission/core"
	"github.com/ethereum/go-ethereum/private"
	"github.com/ethereum/go-ethereum/private/engine"
	"github.com/ethereum/go-ethereum/qlight"
	"github.com/naoina/toml"
	"github.com/urfave/cli/v2"
)

var (
	dumpConfigCommand = &cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(nodeFlags, rpcFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type ethstatsConfig struct {
	URL string `toml:",omitempty"`
}

type gethConfig struct {
	Eth      ethconfig.Config
	Node     node.Config
	Ethstats ethstatsConfig
	Metrics  metrics.Config
}

func loadConfig(file string, cfg *gethConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit, gitDate)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth")
	cfg.WSModules = append(cfg.WSModules, "eth")
	cfg.IPCPath = "geth.ipc"
	return cfg
}

// makeConfigNode loads geth configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	// Quorum: Must occur before setQuorumConfig, as it needs an initialised PTM to be enabled
	// 		   Extension Service and Multitenancy feature validation also depend on PTM availability
	if err := quorumInitialisePrivacy(ctx); err != nil {
		utils.Fatalf("Error initialising Private Transaction Manager: %s", err.Error())
	}

	// Load defaults.
	cfg := gethConfig{
		Eth:     ethconfig.Defaults,
		Node:    defaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}

	// Load config file.
	if file := ctx.String(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	utils.SetQLightConfig(ctx, &cfg.Node, &cfg.Eth)

	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
	}
	applyMetricConfig(ctx, &cfg)

	// Quorum
	if cfg.Eth.QuorumLightServer {
		p2p.SetQLightTLSConfig(readQLightServerTLSConfig(ctx))
		// permissioning for the qlight P2P server
		stack.QServer().SetNewTransportFunc(p2p.NewQlightServerTransport)
		if ctx.IsSet(utils.QuorumLightServerP2PPermissioningFlag.Name) {
			prefix := "qlight"
			if ctx.IsSet(utils.QuorumLightServerP2PPermissioningPrefixFlag.Name) {
				prefix = ctx.String(utils.QuorumLightServerP2PPermissioningPrefixFlag.Name)
			}
			fbp := core.NewFileBasedPermissoningWithPrefix(prefix)
			stack.QServer().SetIsNodePermissioned(fbp.IsNodePermissionedEnode)
		}
	}
	if cfg.Eth.QuorumLightClient.Enabled() {
		p2p.SetQLightTLSConfig(readQLightClientTLSConfig(ctx))
		stack.Server().SetNewTransportFunc(p2p.NewQlightClientTransport)
	}
	// End Quorum

	return stack, cfg
}

// makeFullNode loads geth configuration and creates the Ethereum backend.
func makeFullNode(ctx *cli.Context) (*node.Node, ethapi.Backend) {
	stack, cfg := makeConfigNode(ctx)
	if ctx.IsSet(utils.OverrideBerlinFlag.Name) {
		cfg.Eth.OverrideBerlin = new(big.Int).SetUint64(ctx.Uint64(utils.OverrideBerlinFlag.Name))
	}

	// Quorum: Must occur before registering the extension service, as it needs an initialised PTM to be enabled
	if err := quorumInitialisePrivacy(ctx); err != nil {
		utils.Fatalf("Error initialising Private Transaction Manager: %s", err.Error())
	}

	backend, eth := utils.RegisterEthService(stack, &cfg.Eth)

	// Configure catalyst.
	if ctx.Bool(utils.CatalystFlag.Name) {
		if eth == nil {
			utils.Fatalf("Catalyst does not work in light client mode.")
		}
		if err := catalyst.Register(stack, eth); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Quorum
	// plugin service must be after eth service so that eth service will be stopped gradually if any of the plugin
	// fails to start
	if cfg.Node.Plugins != nil {
		utils.RegisterPluginService(stack, &cfg.Node, ctx.Bool(utils.PluginSkipVerifyFlag.Name), ctx.Bool(utils.PluginLocalVerifyFlag.Name), ctx.String(utils.PluginPublicKeyFlag.Name))
		log.Debug("plugin manager", "value", stack.PluginManager())
		err := eth.NotifyRegisteredPluginService(stack.PluginManager())
		if err != nil {
			utils.Fatalf("Error initialising QLight Token Manager: %s", err.Error())
		}
	}

	if cfg.Node.IsPermissionEnabled() {
		utils.RegisterPermissionService(stack, ctx.Bool(utils.RaftDNSEnabledFlag.Name), backend.ChainConfig().ChainID)
	}

	if ctx.Bool(utils.RaftModeFlag.Name) && !cfg.Eth.QuorumLightClient.Enabled() {
		utils.RegisterRaftService(stack, ctx, &cfg.Node, eth)
	}

	if private.IsQuorumPrivacyEnabled() {
		utils.RegisterExtensionService(stack, eth)
	}
	// End Quorum

	// Configure GraphQL if requested
	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, cfg.Node)
	}
	// Add the Ethereum Stats daemon if requested.
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, backend, cfg.Ethstats.URL)
	}
	return stack, backend
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.Eth.Genesis != nil {
		cfg.Eth.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

func applyMetricConfig(ctx *cli.Context, cfg *gethConfig) {
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		cfg.Metrics.EnabledExpensive = ctx.Bool(utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.IsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.String(utils.MetricsHTTPFlag.Name)
	}
	if ctx.IsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.Int(utils.MetricsPortFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.String(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.String(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.String(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.String(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.String(utils.MetricsInfluxDBTagsFlag.Name)
	}
}

// Quorum

func readQLightClientTLSConfig(ctx *cli.Context) *tls.Config {
	if !ctx.IsSet(utils.QuorumLightTLSFlag.Name) {
		return nil
	}
	if !ctx.IsSet(utils.QuorumLightTLSCACertsFlag.Name) {
		utils.Fatalf("QLight tls flag is set but no client certificate authorities has been provided")
	}
	tlsConfig, err := qlight.NewTLSConfig(&qlight.TLSConfig{
		CACertFileName: ctx.String(utils.QuorumLightTLSCACertsFlag.Name),
		CertFileName:   ctx.String(utils.QuorumLightTLSCertFlag.Name),
		KeyFileName:    ctx.String(utils.QuorumLightTLSKeyFlag.Name),
		ServerName:     enode.MustParse(ctx.String(utils.QuorumLightClientServerNodeFlag.Name)).IP().String(),
		CipherSuites:   ctx.String(utils.QuorumLightTLSCipherSuitesFlag.Name),
	})

	if err != nil {
		utils.Fatalf("Unable to load the specified tls configuration: %v", err)
	}
	return tlsConfig
}

func readQLightServerTLSConfig(ctx *cli.Context) *tls.Config {
	if !ctx.IsSet(utils.QuorumLightTLSFlag.Name) {
		return nil
	}
	if !ctx.IsSet(utils.QuorumLightTLSCertFlag.Name) {
		utils.Fatalf("QLight TLS is enabled but no server certificate has been provided")
	}
	if !ctx.IsSet(utils.QuorumLightTLSKeyFlag.Name) {
		utils.Fatalf("QLight TLS is enabled but no server key has been provided")
	}

	tlsConfig, err := qlight.NewTLSConfig(&qlight.TLSConfig{
		CertFileName:         ctx.String(utils.QuorumLightTLSCertFlag.Name),
		KeyFileName:          ctx.String(utils.QuorumLightTLSKeyFlag.Name),
		ClientCACertFileName: ctx.String(utils.QuorumLightTLSCACertsFlag.Name),
		ClientAuth:           ctx.Int(utils.QuorumLightTLSClientAuthFlag.Name),
		CipherSuites:         ctx.String(utils.QuorumLightTLSCipherSuitesFlag.Name),
	})

	if err != nil {
		utils.Fatalf("QLight TLS - unable to read server tls configuration: %v", err)
	}

	return tlsConfig
}

// quorumValidateEthService checks quorum features that depend on the ethereum service
func quorumValidateEthService(stack *node.Node, isRaft bool) {
	var ethereum *eth.Ethereum

	err := stack.Lifecycle(&ethereum)
	if err != nil {
		utils.Fatalf("Error retrieving Ethereum service: %v", err)
	}

	quorumValidateConsensus(ethereum, isRaft)

	quorumValidatePrivacyEnhancements(ethereum)
}

// quorumValidateConsensus checks if a consensus was used. The node is killed if consensus was not used
func quorumValidateConsensus(ethereum *eth.Ethereum, isRaft bool) {
	transitionAlgorithmOnBlockZero := false
	ethereum.BlockChain().Config().GetTransitionValue(big.NewInt(0), func(transition params.Transition) {
		transitionAlgorithmOnBlockZero = strings.EqualFold(transition.Algorithm, params.IBFT) || strings.EqualFold(transition.Algorithm, params.QBFT)
	})
	if !transitionAlgorithmOnBlockZero && !isRaft && ethereum.BlockChain().Config().Istanbul == nil && ethereum.BlockChain().Config().IBFT == nil && ethereum.BlockChain().Config().QBFT == nil && ethereum.BlockChain().Config().Clique == nil {
		utils.Fatalf("Consensus not specified. Exiting!!")
	}
}

// quorumValidatePrivacyEnhancements checks if privacy enhancements are configured the transaction manager supports
// the PrivacyEnhancements feature
func quorumValidatePrivacyEnhancements(ethereum *eth.Ethereum) {
	privacyEnhancementsBlock := ethereum.BlockChain().Config().PrivacyEnhancementsBlock

	for _, transition := range ethereum.BlockChain().Config().Transitions {
		if transition.PrivacyPrecompileEnabled != nil && *transition.PrivacyEnhancementsEnabled {
			privacyEnhancementsBlock = transition.Block
			break
		}
	}

	if privacyEnhancementsBlock != nil {
		log.Info("Privacy enhancements is configured to be enabled from block ", "height", privacyEnhancementsBlock)
		if !private.P.HasFeature(engine.PrivacyEnhancements) {
			utils.Fatalf("Cannot start quorum with privacy enhancements enabled while the transaction manager does not support it")
		}
	}
}

// configure and set up quorum transaction privacy
func quorumInitialisePrivacy(ctx *cli.Context) error {
	cfg, err := QuorumSetupPrivacyConfiguration(ctx)
	if err != nil {
		return err
	}

	err = private.InitialiseConnection(cfg, ctx.IsSet(utils.QuorumLightClientFlag.Name))
	if err != nil {
		return err
	}
	privacyExtension.Init()

	return nil
}

// Get private transaction manager configuration
func QuorumSetupPrivacyConfiguration(ctx *cli.Context) (http.Config, error) {
	// get default configuration
	cfg, err := private.GetLegacyEnvironmentConfig()
	if err != nil {
		return http.Config{}, err
	}

	// override the config with command line parameters
	if ctx.IsSet(utils.QuorumPTMUnixSocketFlag.Name) {
		cfg.SetSocket(ctx.String(utils.QuorumPTMUnixSocketFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMUrlFlag.Name) {
		cfg.SetHttpUrl(ctx.String(utils.QuorumPTMUrlFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMTimeoutFlag.Name) {
		cfg.SetTimeout(ctx.Uint(utils.QuorumPTMTimeoutFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMDialTimeoutFlag.Name) {
		cfg.SetDialTimeout(ctx.Uint(utils.QuorumPTMDialTimeoutFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMHttpIdleTimeoutFlag.Name) {
		cfg.SetHttpIdleConnTimeout(ctx.Uint(utils.QuorumPTMHttpIdleTimeoutFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMHttpWriteBufferSizeFlag.Name) {
		cfg.SetHttpWriteBufferSize(ctx.Int(utils.QuorumPTMHttpWriteBufferSizeFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMHttpReadBufferSizeFlag.Name) {
		cfg.SetHttpReadBufferSize(ctx.Int(utils.QuorumPTMHttpReadBufferSizeFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMTlsModeFlag.Name) {
		cfg.SetTlsMode(ctx.String(utils.QuorumPTMTlsModeFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMTlsRootCaFlag.Name) {
		cfg.SetTlsRootCA(ctx.String(utils.QuorumPTMTlsRootCaFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMTlsClientCertFlag.Name) {
		cfg.SetTlsClientCert(ctx.String(utils.QuorumPTMTlsClientCertFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMTlsClientKeyFlag.Name) {
		cfg.SetTlsClientKey(ctx.String(utils.QuorumPTMTlsClientKeyFlag.Name))
	}
	if ctx.IsSet(utils.QuorumPTMTlsInsecureSkipVerify.Name) {
		cfg.SetTlsInsecureSkipVerify(ctx.Bool(utils.QuorumPTMTlsInsecureSkipVerify.Name))
	}

	if err = cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
