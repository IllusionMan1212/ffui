package main

import "fmt"

var Configs = []Config{
	{Name: "Delete old video(s)?", Opts: []string{"No", "Yes"}, FocusedOption: 1},
	{Name: "What should we do about encoded videos?", Opts: []string{"Skip", "Delete and encode again"}},
	{Name: "Video Encoder", Opts: []string{"copy", "libx264", "libx265", "libvpx-vp9", "librav1e", "libsvtav1"}},
	{Name: "Audio Encoder", Opts: []string{"None", "copy", "aac", "libopus"}, FocusedOption: 1},
	{Name: "Preset", Opts: []string{"ultrafast", "superfast", "veryfast", "faster", "fast", "medium", "slow", "slower", "veryslow"}, FocusedOption: 4},
	{Name: "Constant Rate Factor (CRF)", Opts: []string{"10", "15", "20", "25", "30", "35", "40", "45", "50"}, FocusedOption: 4},
}

type Config struct {
	Name          string
	Opts          []string
	FocusedOption int
}

type ParsedConfig struct {
	DeleteOldVideo bool
	SkipEncodedVid bool
	VideoEncoder   string
	AudioEncoder   string
	Preset         string
	CRF            string
}

func find(cfgs []Config, name string) Config {
	for _, cfg := range cfgs {
		if cfg.Name == name {
			return cfg
		}
	}

	err := fmt.Sprintf("Couldn't find requested cfg: %s", name)
	panic(err)
}

func filter(cfgs []Config, names ...string) []Config {
	filtered := make([]Config, 0)

cfgloop:
	for _, cfg := range cfgs {
		for _, name := range names {
			if cfg.Name == name {
				continue cfgloop
			}
		}

		filtered = append(filtered, cfg)
	}

	return filtered
}

func getVisibleConfigs(cfgs []Config) []Config {
	parsed := parseConfig(cfgs)

	switch parsed.VideoEncoder {
	case "copy", "librav1e":
		return filter(cfgs, "Preset", "Constant Rate Factor (CRF)")
	case "libvpx-vp9", "libsvtav1":
		return filter(cfgs, "Preset")
	}

	return cfgs
}
