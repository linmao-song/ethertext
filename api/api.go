package api

import (
	"context"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/sirupsen/logrus"
	"github.com/songlinm/ethertext/blockreader"
)

type Server struct {
	blockreader *blockreader.Blockreader
	server      *http.Server
}

func NewServer(chain *core.BlockChain, listenAddr string, cacheSize int) *Server {
	b := blockreader.NewBlockReader(chain, cacheSize)
	router := http.NewServeMux()
	router.Handle("/", logRequest(index()))
	router.Handle("/text", logRequest(ethertext(b)))
	router.Handle("/start", logRequest(start()))

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return &Server{
		blockreader: b,
		server:      server,
	}
}

func (s *Server) Start(ctx context.Context) {
	_, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := s.server.ListenAndServe(); err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Closing down server...")
			s.server.Close()
			return
		}
	}
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		w.Write(getPage(46214))
	})
}

func start() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var blkNum uint64
		var err error
		if num, exists := q["blocknum"]; exists {
			if len(num) != 1 {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if blkNum, err = strconv.ParseUint(num[0], 10, 64); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		w.Write(getPage(blkNum))
		return
	})
}

func ethertext(b *blockreader.Blockreader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var blkNum uint64
		var err error
		if num, exists := q["blocknum"]; exists {
			if len(num) != 1 {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if blkNum, err = strconv.ParseUint(num[0], 10, 64); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}
		json := b.Get(blkNum, 100)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
		logrus.Infof("finished processing ")
		return
	})
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := httputil.DumpRequest(r, true)
		if err != nil {
			logrus.WithError(err).Warn("Failed to dump request")
		}
		logrus.WithFields(logrus.Fields{
			"from":   r.RemoteAddr,
			"method": r.Method,
			"url":    r.URL,
			"req":    string(raw),
		}).Info("Processing request")
		handler.ServeHTTP(w, r)
	})
}
