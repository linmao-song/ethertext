package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/params"
	"github.com/sirupsen/logrus"
	"github.com/songlinm/ethertext/api"
)

const clientIdentifier = "Ethertext"

var (
	dataDir    string
	listenAddr string
	logFile    string
	cacheSize  int
)

type gethConfig struct {
	Eth  eth.Config
	Node node.Config
}

func init() {
	flag.StringVar(&dataDir, "datadir", "", "data storage directory")
	flag.StringVar(&listenAddr, "listenaddr", "", "http server address <host:port>")
	flag.StringVar(&logFile, "logfile", "", "file to write logs to")
	flag.IntVar(&cacheSize, "cachesize", 1024*1024*10, "number of entries in cache")
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit("0.0.1")
	cfg.IPCPath = "ethtext.ipc"

	if len(dataDir) > 0 {
		logrus.Info("Using data directory ", dataDir)
		cfg.DataDir = dataDir
	}

	return cfg
}

func makeConfigNode() (*node.Node, gethConfig) {
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,
		Node: defaultNodeConfig(),
	}

	cfg.Node.P2P.DiscoveryV5 = true
	cfg.Node.P2P.BootstrapNodes = make([]*discover.Node, 0, len(params.MainnetBootnodes))
	for _, url := range params.MainnetBootnodes {
		node, _ := discover.ParseNode(url)
		cfg.Node.P2P.BootstrapNodes = append(cfg.Node.P2P.BootstrapNodes, node)
	}

	stack, err := node.New(&cfg.Node)
	if err != nil {
		logrus.WithError(err).Panic("Failed to create the protocol stack")
	}

	return stack, cfg
}

func makeFullNode() (*node.Node, *core.BlockChain) {
	stack, cfg := makeConfigNode()

	var le *eth.Ethereum
	var err error

	lock := &sync.Mutex{}
	lock.Lock()
	cond := sync.NewCond(lock)

	logrus.Info("registering LES")
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		logrus.Info("creating LES")
		le, err = eth.New(ctx, &cfg.Eth)
		logrus.Info("LES created")
		cond.Signal()
		return le, err
	})

	utils.StartNode(stack)

	for le == nil {
		cond.Wait()
	}
	lock.Unlock()
	logrus.Info("NODE created")
	return stack, le.BlockChain()
}

func etherText(ctx context.Context) {
	node, chain := makeFullNode()

	ch := make(chan core.ChainEvent)
	sub := chain.SubscribeChainEvent(ch)
	logrus.Infof("Chain event subscribed %v", sub)

	if len(listenAddr) == 0 {
		listenAddr = "localhost:80"
	}
	s := api.NewServer(chain, listenAddr, cacheSize)
	go s.Start(ctx)

	for {
		select {
		case ev := <-ch:
			logrus.Infof("Chain event %v", ev)
		case er := <-sub.Err():
			logrus.Infof("Err event %v", er)
		case <-ctx.Done():
			logrus.Info("Existing...")
			node.Stop()
			return
		}
	}
}

func main() {
	flag.Parse()

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logrus.SetFormatter(formatter)

	if len(logFile) > 0 {
		if log, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE, 0744); err != nil {
			logrus.WithError(err).Panic("Error creating log file")
		} else {
			logrus.SetOutput(log)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	etherText(ctx)
}
