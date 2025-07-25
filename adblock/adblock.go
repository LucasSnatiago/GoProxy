package adblock

import (
	"bufio"
	"log"
	"strings"

	"github.com/LucasSnatiago/GoProxy/pac"
	"github.com/LucasSnatiago/GoProxy/proxyhandler"
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

func DownloadStevensBlackBlackList(pacparser *pac.Pac) *AdBlocker {
	data, err := proxyhandler.GetBytesFromURL("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts", pacparser)
	if err != nil || len(data) == 0 {
		log.Printf("Failed to download Stevens Black List: %v\nTurning adblock off", err)
	}

	entries, err := parseHostList(bufio.NewScanner(strings.NewReader(string(data))))
	if err != nil || len(entries) == 0 {
		log.Printf("Failed to parse Stevens Black List: %v\nTurning adblock off", err)
	}

	return &AdBlocker{
		Entries: entries,
	}
}

func parseHostList(scanner *bufio.Scanner) (map[string]string, error) {
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
