package status

import (
	"runtime/debug"
)

type BuildInfo struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

func buildInfo() *BuildInfo {
	info, ok := debug.ReadBuildInfo()

	if !ok {
		return nil
	}

	buildInfo := new(BuildInfo)
	buildInfo.Name = info.Main.Path
	buildInfo.Version = gitRevision(info.Settings)

	return buildInfo
}

func gitRevision(settings []debug.BuildSetting) string {
	for _, setting := range settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}

	return "n/a"
}
