package geoip

import "testing"

func TestRegionWithoutDatabase(t *testing.T) {
	r, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for ip, want := range map[string]string{"127.0.0.1": "本地网络", "192.168.1.2": "本地网络", "::1": "本地网络", "8.8.8.8": "未知", "not-an-ip": "未知"} {
		if got := r.Region(ip); got != want {
			t.Errorf("%s: got %q, want %q", ip, got, want)
		}
	}
}

func TestLocalizedName(t *testing.T) {
	if got := localized(map[string]string{"zh-CN": "上海", "en": "Shanghai"}); got != "上海" {
		t.Fatal(got)
	}
	if got := localized(map[string]string{"en": "Berlin"}); got != "Berlin" {
		t.Fatal(got)
	}
}
