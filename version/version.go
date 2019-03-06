package version

import (
    "strings"
)

// pick up version from build process, but use these defaults
var Version = "v0.4.X"
var Release = "1"
var GitHash = "0000000"
var LongVersion = ""
var Descriptor = ""

type VersionInfo struct {
    Version     string
    LongVersion string
    GitHash     string
}

func GetVersion() string {
    return strings.TrimSpace(Version)
}

func GetLongVersion() string {
    // allow LongVersion override
    if "" != LongVersion {
        return LongVersion
    }
    // add git hash to long version if available
    version := GetVersion()
    if "" != GetRelease() {
        version = version + "-" + GetRelease()
    }
    if "" != GetGitHash() {
        version = version + "-git@" + GetGitHash()
    }
    if "" != GetDescriptor() {
        version = version + " (" + GetDescriptor() + ")"
    }

    return strings.TrimSpace(version)
}

func GetGitHash() string {
    return strings.TrimSpace(GitHash)
}

func GetRelease() string {
    return strings.TrimSpace(Release)
}

func GetDescriptor() string {
    return strings.TrimSpace(Descriptor)    
}

func Info() VersionInfo {
    return VersionInfo {
        Version:     GetVersion(),
        LongVersion: GetLongVersion(),
        GitHash:     GetGitHash(),
    }
}