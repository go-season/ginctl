package upgrade

import (
	"errors"
	"regexp"
	"sync"

	"github.com/blang/semver"
	"github.com/go-season/go-selfupdate/selfupdate"
)

var version string
var rawVersion string

var gitlabSlug = "efficacy-org/ginctl"
var reVersion = regexp.MustCompile(`\d+\.\d+\.\d+`)

func GetVersion() string {
	return version
}

func SetVersion(verText string) {
	if len(verText) > 0 {
		_version, err := eraseVersionPrefix(verText)
		if err != nil {
			// write log
			return
		}

		version = _version
		rawVersion = verText
	}
}

func eraseVersionPrefix(version string) (string, error) {
	indices := reVersion.FindStringIndex(version)
	if indices == nil {
		return version, errors.New("version not adopting semver")
	}
	if indices[0] > 0 {
		version = version[indices[0]:]
	}

	return version, nil
}

func NewVersionAvailable() string {
	version := GetVersion()
	if version != "" {
		latestStableVersion, err := CheckForNewerVersion()
		if latestStableVersion != "" && err == nil {
			semverVersion, err := semver.Parse(version)
			if err == nil {
				semverLatestStableVersion, err := semver.Parse(latestStableVersion)
				if err == nil {
					if semverLatestStableVersion.Compare(semverVersion) == 1 {
						return latestStableVersion
					}
				}
			}
		}
	}

	return ""
}

var (
	latestVersion     string
	latestVersionErr  error
	latestVersionOnce sync.Once
)

func CheckForNewerVersion() (string, error) {
	latestVersionOnce.Do(func() {
		latest, found, err := selfupdate.DetectLatest(gitlabSlug)
		if err != nil {
			latestVersionErr = err
			return
		}

		v := semver.MustParse(version)
		if !found || latest.Version.Equals(v) {
			return
		}

		latestVersion = latest.Version.String()
	})

	return latestVersion, latestVersionErr
}

func Upgrade() error {
	// log

	newerVersion, err := CheckForNewerVersion()
	if err != nil {
		return err
	}
	if newerVersion == "" {
		//log
		return nil
	}

	return nil
}
