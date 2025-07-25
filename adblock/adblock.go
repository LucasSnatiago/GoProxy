package adblock

import (
	"bufio"
	"log"
	"strings"

	"github.com/LucasSnatiago/GoProxy/pac"
)

type AdBlocker struct {
	Entries map[string]string
}

func NewAdblock(pacparser *pac.Pac) *AdBlocker {
	adblock := DownloadStevensBlackBlackList(pacparser)
	if len(adblock.Entries) == 0 {
		log.Println("AdBlock is disabled, no entries found.")
	}

	return adblock
}

func ParseHostList(scanner *bufio.Scanner) (map[string]string, error) {
	tmp := make(map[string]string)
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
		host := fields[1]
		tmp[host] = fields[0]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tmp, nil
}

func (a *AdBlocker) CheckIfAppearsOnAdblockList(host string) bool {
	_, ok := a.Entries[host]
	return ok
}
