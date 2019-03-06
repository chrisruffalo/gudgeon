package version

// pick up version from build process, but use these defaults
var Version = "v0.3.X"
var GitHash = "0000000"
var LongVersion = ""

type VersionInfo struct {
    Version     string
    LongVersion string
    GitHash     string
}

func GetVersion() string {
    return Version
}

func GetLongVersion() string {
    // allow LongVersion override
    if "" != LongVersion {
        return LongVersion
    }
    // add git hash to long version if available
    if "" != GitHash {
        return Version + "@git" + GitHash
    }
    return GetVersion()
}

func GetGitHash() string {
    return GitHash
}

func Info() VersionInfo {
    return VersionInfo {
        Version:     GetVersion(),
        LongVersion: GetLongVersion(),
        GitHash:     GetGitHash(),
    }
}