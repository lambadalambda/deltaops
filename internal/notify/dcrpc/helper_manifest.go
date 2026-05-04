package dcrpc

const HelperReleaseVersion = "v2.49.0"

const helperReleaseBaseURL = "https://github.com/chatmail/core/releases/download/" + HelperReleaseVersion

type HelperReleaseAsset struct {
	Version  string
	GOOS     string
	GOARCH   string
	Filename string
	SHA256   string
	URL      string
}

var helperReleaseAssets = []HelperReleaseAsset{
	helperReleaseAsset("linux", "amd64", "deltachat-rpc-server-x86_64-linux", "28e10b40518f55fa8ce20edd119fa743dd29a22df372b58443ec53eb753cb50c"),
	helperReleaseAsset("linux", "arm64", "deltachat-rpc-server-aarch64-linux", "33acdc048060fcd51bc585f2eefdaa2cf93cca9306440f45be8c5936024732cf"),
	helperReleaseAsset("linux", "386", "deltachat-rpc-server-i686-linux", "6fe6831f0bcd84316dafa416883249aba623eb392b7795769d7b9f635dc069b6"),
	helperReleaseAsset("darwin", "arm64", "deltachat-rpc-server-aarch64-macos", "3ea30551ddaa67c2691c1cfbf0087ad95b799c5192269aada232ca2569891789"),
	helperReleaseAsset("darwin", "amd64", "deltachat-rpc-server-x86_64-macos", "a8885769dc24eacd605b32593332de138fc77d97550b709c330d4fd4479b48c9"),
}

func helperReleaseAsset(goos, goarch, filename, sha256 string) HelperReleaseAsset {
	return HelperReleaseAsset{
		Version:  HelperReleaseVersion,
		GOOS:     goos,
		GOARCH:   goarch,
		Filename: filename,
		SHA256:   sha256,
		URL:      helperReleaseBaseURL + "/" + filename,
	}
}

func HelperReleaseAssets() []HelperReleaseAsset {
	assets := make([]HelperReleaseAsset, len(helperReleaseAssets))
	copy(assets, helperReleaseAssets)
	return assets
}

func helperSHA256(filename string) string {
	for _, asset := range helperReleaseAssets {
		if asset.Filename == filename {
			return asset.SHA256
		}
	}
	return ""
}

func SupportedHelperTargets() []string {
	targets := make([]string, 0, len(helperReleaseAssets))
	for _, asset := range helperReleaseAssets {
		targets = append(targets, asset.GOOS+"/"+asset.GOARCH)
	}
	return targets
}
