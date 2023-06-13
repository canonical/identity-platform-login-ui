package health

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
)

const okValue = "ok"

type Status struct {
	Status    string     `json:"status"`
	BuildInfo *BuildInfo `json:"buildInfo"`
}

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

func HandleAlive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	rr := Status{
		Status: okValue,
	}

	if buildInfo := buildInfo(); buildInfo != nil {
		rr.BuildInfo = buildInfo
	}

	json.NewEncoder(w).Encode(rr)

	return
}
