package main

import (
	"fmt"
	"runtime/debug"
)

var (
	version    string
	revision   string
	buildDirty bool
	buildTime  string
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if version == "" {
		version = info.Main.Version
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value[:7]
		}
		if setting.Key == "vcs.modified" {
			buildDirty = setting.Value == "true"
		}
		if setting.Key == "vcs.time" {
			buildTime = setting.Value
		}
	}
}

func Ternary[T any](cond bool, t, f T) T {
	if cond {
		return t
	} else {
		return f
	}
}

func GetFormattedVersion() string {
	revisionMeta := revision +
		Ternary(buildDirty, "+dirty", "") +
		Ternary(buildTime != "", ", "+buildTime, "")
	if revisionMeta == "" {
		return version
	} else {
		return fmt.Sprintf("%s (%s)", version, revisionMeta)
	}
}
