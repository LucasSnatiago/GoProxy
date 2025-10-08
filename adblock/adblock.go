package adblock

import (
	"bufio"
	"log"
	"slices"
	"strings"

	"github.com/LucasSnatiago/GoProxy/pac"
)

type AdBlocker struct {
	Entries []string
}

func NewAdblock(adblockUrl string, pacparser *pac.Pac) *AdBlocker {
	adblock := DownloadStevensBlackBlackList(adblockUrl, pacparser)
	if len(adblock.Entries) == 0 {
		log.Println("AdBlock is disabled, no entries found.")
	}

	return adblock
}

func ParseHostList(scanner *bufio.Scanner) ([]string, error) {
	var tmp []string
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
		tmp = append(tmp, fields[1])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tmp, nil
}

func (a *AdBlocker) CheckIfAppearsOnAdblockList(host string) bool {
	return slices.Contains(a.Entries, host)
}
