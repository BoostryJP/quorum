// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package debug

import (
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var (
	verbosityFlag = &cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	vmoduleFlag = &cli.StringFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value: "",
	}
	logjsonFlag = &cli.BoolFlag{
		Name:  "log.json",
		Usage: "Format logs with JSON",
	}
	backtraceAtFlag = &cli.StringFlag{
		Name:  "log.backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\")",
		Value: "",
	}
	debugFlag = &cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
	pprofFlag = &cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the pprof HTTP server",
	}
	pprofPortFlag = &cli.IntFlag{
		Name:  "pprof.port",
		Usage: "pprof HTTP server listening port",
		Value: 6060,
	}
	pprofAddrFlag = &cli.StringFlag{
		Name:  "pprof.addr",
		Usage: "pprof HTTP server listening interface",
		Value: "127.0.0.1",
	}
	memprofilerateFlag = &cli.IntFlag{
		Name:  "pprof.memprofilerate",
		Usage: "Turn on memory profiling with the given rate",
		Value: runtime.MemProfileRate,
	}
	blockprofilerateFlag = &cli.IntFlag{
		Name:  "pprof.blockprofilerate",
		Usage: "Turn on block profiling with the given rate",
	}
	cpuprofileFlag = &cli.StringFlag{
		Name:  "pprof.cpuprofile",
		Usage: "Write CPU profile to the given file",
	}
	traceFlag = &cli.StringFlag{
		Name:  "trace",
		Usage: "Write execution trace to the given file",
	}
	// (Deprecated April 2020)
	legacyPprofPortFlag = &cli.IntFlag{
		Name:  "pprofport",
		Usage: "pprof HTTP server listening port (deprecated, use --pprof.port)",
		Value: 6060,
	}
	legacyPprofAddrFlag = &cli.StringFlag{
		Name:  "pprofaddr",
		Usage: "pprof HTTP server listening interface (deprecated, use --pprof.addr)",
		Value: "127.0.0.1",
	}
	legacyMemprofilerateFlag = &cli.IntFlag{
		Name:  "memprofilerate",
		Usage: "Turn on memory profiling with the given rate (deprecated, use --pprof.memprofilerate)",
		Value: runtime.MemProfileRate,
	}
	legacyBlockprofilerateFlag = &cli.IntFlag{
		Name:  "blockprofilerate",
		Usage: "Turn on block profiling with the given rate (deprecated, use --pprof.blockprofilerate)",
	}
	legacyCpuprofileFlag = &cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "Write CPU profile to the given file (deprecated, use --pprof.cpuprofile)",
	}
	legacyBacktraceAtFlag = &cli.StringFlag{
		Name:  "backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\") (deprecated, use --log.backtrace)",
		Value: "",
	}
	legacyDebugFlag = &cli.BoolFlag{
		Name:  "debug",
		Usage: "Prepends log messages with call-site location (file and line number) (deprecated, use --log.debug)",
	}
)

// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag,
	vmoduleFlag,
	logjsonFlag,
	backtraceAtFlag,
	debugFlag,
	pprofFlag,
	pprofAddrFlag,
	pprofPortFlag,
	memprofilerateFlag,
	blockprofilerateFlag,
	cpuprofileFlag,
	traceFlag,
}

// This is the list of deprecated debugging flags.
var DeprecatedFlags = []cli.Flag{
	legacyPprofPortFlag,
	legacyPprofAddrFlag,
	legacyMemprofilerateFlag,
	legacyBlockprofilerateFlag,
	legacyCpuprofileFlag,
	legacyBacktraceAtFlag,
	legacyDebugFlag,
}

var glogger *log.GlogHandler

func init() {
	glogger = log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)
}

// Setup initializes profiling and logging based on the CLI flags.
// It should be called as early as possible in the program.
func Setup(ctx *cli.Context) error {
	var ostream log.Handler
	output := io.Writer(os.Stderr)
	if ctx.Bool(logjsonFlag.Name) {
		ostream = log.StreamHandler(output, log.JSONFormat())
	} else {
		usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		if usecolor {
			output = colorable.NewColorableStderr()
		}
		ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))
	}
	glogger.SetHandler(ostream)
	// logging
	verbosity := ctx.Int(verbosityFlag.Name)
	glogger.Verbosity(log.Lvl(verbosity))
	vmodule := ctx.String(vmoduleFlag.Name)
	glogger.Vmodule(vmodule)

	debug := ctx.Bool(debugFlag.Name)
	if ctx.IsSet(legacyDebugFlag.Name) {
		debug = ctx.Bool(legacyDebugFlag.Name)
		log.Warn("The flag --debug is deprecated and will be removed in the future, please use --log.debug")
	}
	if ctx.IsSet(debugFlag.Name) {
		debug = ctx.Bool(debugFlag.Name)
	}
	log.PrintOrigins(debug)

	backtrace := ctx.String(backtraceAtFlag.Name)
	if b := ctx.String(legacyBacktraceAtFlag.Name); b != "" {
		backtrace = b
		log.Warn("The flag --backtrace is deprecated and will be removed in the future, please use --log.backtrace")
	}
	if b := ctx.String(backtraceAtFlag.Name); b != "" {
		backtrace = b
	}
	glogger.BacktraceAt(backtrace)

	log.Root().SetHandler(glogger)

	// profiling, tracing
	runtime.MemProfileRate = memprofilerateFlag.Value
	if ctx.IsSet(legacyMemprofilerateFlag.Name) {
		runtime.MemProfileRate = ctx.Int(legacyMemprofilerateFlag.Name)
		log.Warn("The flag --memprofilerate is deprecated and will be removed in the future, please use --pprof.memprofilerate")
	}
	if ctx.IsSet(memprofilerateFlag.Name) {
		runtime.MemProfileRate = ctx.Int(memprofilerateFlag.Name)
	}

	blockProfileRate := blockprofilerateFlag.Value
	if ctx.IsSet(legacyBlockprofilerateFlag.Name) {
		blockProfileRate = ctx.Int(legacyBlockprofilerateFlag.Name)
		log.Warn("The flag --blockprofilerate is deprecated and will be removed in the future, please use --pprof.blockprofilerate")
	}
	if ctx.IsSet(blockprofilerateFlag.Name) {
		blockProfileRate = ctx.Int(blockprofilerateFlag.Name)
	}
	Handler.SetBlockProfileRate(blockProfileRate)

	if traceFile := ctx.String(traceFlag.Name); traceFile != "" {
		if err := Handler.StartGoTrace(traceFile); err != nil {
			return err
		}
	}

	if cpuFile := ctx.String(cpuprofileFlag.Name); cpuFile != "" {
		if err := Handler.StartCPUProfile(cpuFile); err != nil {
			return err
		}
	}

	// pprof server
	if ctx.Bool(pprofFlag.Name) {
		listenHost := ctx.String(pprofAddrFlag.Name)

		port := ctx.Int(pprofPortFlag.Name)

		address := fmt.Sprintf("%s:%d", listenHost, port)
		// This context value ("metrics.addr") represents the utils.MetricsHTTPFlag.Name.
		// It cannot be imported because it will cause a cyclical dependency.
		StartPProf(address, !ctx.IsSet("metrics.addr"))
	}
	return nil
}

func StartPProf(address string, withMetrics bool) {
	// Hook go-metrics into expvar on any /debug/metrics request, load all vars
	// from the registry into expvar, and execute regular expvar handler.
	if withMetrics {
		exp.Exp(metrics.DefaultRegistry)
	}
	log.Info("Starting pprof server", "addr", fmt.Sprintf("http://%s/debug/pprof", address))
	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Error("Failure in running pprof server", "err", err)
		}
	}()
}

// Exit stops all running profiles, flushing their output to the
// respective file.
func Exit() {
	Handler.StopCPUProfile()
	Handler.StopGoTrace()
}
