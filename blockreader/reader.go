package blockreader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
)

var (
	blackList = map[rune]bool{
		rune('\xdb'):   true,
		rune('\xef'):   true,
		rune('\xbf'):   true,
		rune('\xbd'):   true,
		rune('\x8e'):   true,
		rune('\xcf'):   true,
		rune('\x00'):   true,
		rune('\ufffd'): true,
	}
)

type TextRec struct {
	Text string
	Txn  string
}

type TextInBlock struct {
	BlockNum uint64
	Text     []*TextRec
}

func (t *TextInBlock) Json() (ret string) {
	js, _ := json.Marshal(t)
	return string(js)
}

func include(r rune) bool {
	if unicode.IsPrint(r) {
		if _, exists := blackList[r]; !exists {
			return true
		}
	}
	return false
}

type Blockreader struct {
	chain *core.BlockChain
	cache *lru.ARCCache // num => json string
}

func NewBlockReader(chain *core.BlockChain, cacheSize int) *Blockreader {
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		logrus.WithError(err).Panic("Failed to create ARC cache")
	}
	return &Blockreader{
		chain: chain,
		cache: cache,
	}
}

func (b *Blockreader) Get(startBlock uint64, num uint64) (ret []byte) {
	logrus.Infof("Getting %d blocks: starting from %d", num, startBlock)
	var jsons []string
	for n := startBlock; num > 0; n++ {
		if t, ok := b.cache.Get(n); ok {
			logrus.Debugf("block %d found in cache", n)
			json := t.(string)
			jsons = append(jsons, json)
			num--
			continue
		}
		if txts, err := b.getBlock(n); err == nil && len(txts) > 0 {
			tb := &TextInBlock{
				BlockNum: n,
				Text:     txts,
			}
			json := tb.Json()
			b.cache.Add(n, json)
			jsons = append(jsons, json)
			num--
		} else if err != nil {
			logrus.WithError(err).Warn("stopping search")
			break
		}
	}
	logrus.Infof("Blocks obtained")

	var builder bytes.Buffer
	builder.WriteString(`[`)
	for i, j := range jsons {
		builder.WriteString(j)
		if i != len(jsons)-1 {
			builder.WriteString(",")
		}
	}
	builder.WriteString(`]`)
	logrus.Infof("finished writing")
	return builder.Bytes()
}

func readBlock(blk *types.Block) (ret []*TextRec) {
	for _, tx := range blk.Transactions() {
		data := tx.Data()
		var rs []rune
		for len(data) > 0 {
			r, size := utf8.DecodeRune(data)
			if include(r) {
				rs = append(rs, r)
			}
			data = data[size:]
		}
		if len(rs) > 0 {
			ret = append(ret, &TextRec{
				Text: string(rs),
				Txn:  tx.Hash().Hex()})
		}
	}
	return
}

func (b *Blockreader) getBlock(blkNum uint64) (ret []*TextRec, err error) {
	if blk := b.chain.GetBlockByNumber(blkNum); blk != nil {
		ret = readBlock(blk)
	} else {
		err = fmt.Errorf("No record for block %d", blkNum)
	}
	return
}
