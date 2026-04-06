package adblock

import (
	"bufio"
	"fmt"
	"log"
	"strings"

	"github.com/LucasSnatiago/GoProxy/pac"
	iradix "github.com/hashicorp/go-immutable-radix/v2"
)

type AdBlocker struct {
	Entries        *iradix.Tree[bool]
	cachedToString string
}

func NewAdblock(adblockUrl string, pacparser *pac.Pac) *AdBlocker {
	adblock := DownloadStevensBlackBlackList(adblockUrl, pacparser)
	if adblock.Entries.Len() == 0 {
		log.Println("AdBlock is disabled, no entries found.")
	}

	return adblock
}

func ParseHostList(scanner *bufio.Scanner) (*iradix.Tree[bool], error) {
	r := iradix.New[bool]()
	txn := r.Txn()

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}
		// format: 0.0.0.0 some.domain.com
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		txn.Insert([]byte(fields[1]), true)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	r = txn.Commit()
	return r, nil
}

func (a *AdBlocker) CheckIfAppearsOnAdblockList(host string) bool {
	_, found := a.Entries.Get([]byte(host))
	return found
}

func (a *AdBlocker) ToString() string {
	if a.cachedToString != "" {
		return a.cachedToString
	}

	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("%d entries:\n", a.Entries.Len()))
	for host, _ := range a.Entries.Root().Walk {
		str.WriteString(fmt.Sprintf("%s\n", host))
	}
	a.cachedToString = str.String()

	return a.cachedToString
}
