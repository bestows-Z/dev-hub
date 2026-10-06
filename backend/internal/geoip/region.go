package geoip

import (
	"fmt"
	"net"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

// Resolver only returns a broad region. Raw IP addresses are never stored.
type Resolver struct{ reader *maxminddb.Reader }

func Open(path string) (*Resolver, error) {
	if path == "" {
		return &Resolver{}, nil
	}
	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open IP region database: %w", err)
	}
	return &Resolver{reader: reader}, nil
}

func (r *Resolver) Close() error {
	if r.reader != nil {
		return r.reader.Close()
	}
	return nil
}

func (r *Resolver) Region(rawIP string) string {
	ip := net.ParseIP(strings.TrimSpace(rawIP))
	if ip == nil {
		return "未知"
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return "本地网络"
	}
	if r == nil || r.reader == nil {
		return "未知"
	}
	var record struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		Subdivisions []struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"subdivisions"`
	}
	if err := r.reader.Lookup(ip, &record); err != nil {
		return "未知"
	}
	country := localized(record.Country.Names)
	if record.Country.ISOCode == "CN" {
		if len(record.Subdivisions) > 0 {
			if province := localized(record.Subdivisions[0].Names); province != "" {
				return "中国 · " + province
			}
		}
		return "中国"
	}
	if country != "" {
		return country
	}
	return "未知"
}

func localized(names map[string]string) string {
	if value := names["zh-CN"]; value != "" {
		return value
	}
	return names["en"]
}
