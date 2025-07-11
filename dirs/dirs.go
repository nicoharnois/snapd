// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package dirs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/snapcore/snapd/release"
	"github.com/snapcore/snapd/strutil"
)

// the various file paths
var (
	GlobalRootDir string = "/"

	RunDir string

	SnapMountDir string

	DistroLibExecDir string

	hiddenSnapDataHomeGlob []string

	SnapBlobDir          string
	SnapDataDir          string
	snapDataHomeGlob     []string
	SnapDownloadCacheDir string
	SnapAppArmorDir      string
	SnapLdconfigDir      string
	SnapSeccompBase      string
	SnapSeccompDir       string
	SnapMountPolicyDir   string
	SnapCgroupPolicyDir  string
	SnapUdevRulesDir     string
	SnapKModModulesDir   string
	SnapKModModprobeDir  string
	LocaleDir            string
	SnapdSocket          string
	SnapSocket           string
	SnapRunDir           string
	SnapRunNsDir         string
	SnapRunLockDir       string
	SnapBootstrapRunDir  string
	SnapVoidDir          string

	SnapInterfacesRequestsRunDir   string
	SnapInterfacesRequestsStateDir string

	SnapdMaintenanceFile string

	SnapdStoreSSLCertsDir string

	SnapSeedDir   string
	SnapDeviceDir string

	SnapAssertsDBDir      string
	SnapCookieDir         string
	SnapTrustedAccountKey string
	SnapAssertsSpoolDir   string
	SnapSeqDir            string

	SnapStateFile     string
	SnapStateLockFile string
	SnapSystemKeyFile string

	SnapRepairConfigFile string
	SnapRepairDir        string
	SnapRepairStateFile  string
	SnapRepairRunDir     string
	SnapRepairAssertsDir string
	SnapRunRepairDir     string

	SnapRollbackDir string

	SnapCacheDir        string
	SnapNamesFile       string
	SnapSectionsFile    string
	SnapCommandsDB      string
	SnapAuxStoreInfoDir string
	SnapIconsPoolDir    string
	SnapIconsDir        string

	SnapBinariesDir        string
	SnapServicesDir        string
	SnapRuntimeServicesDir string
	SnapUserServicesDir    string
	SnapSystemdConfDir     string
	SnapDesktopFilesDir    string
	SnapDesktopIconsDir    string
	SnapPolkitPolicyDir    string
	SnapPolkitRuleDir      string
	SnapSystemdDir         string
	SnapSystemdRunDir      string

	SnapDBusSessionPolicyDir   string
	SnapDBusSystemPolicyDir    string
	SnapDBusSessionServicesDir string
	SnapDBusSystemServicesDir  string

	SnapModeenvFile   string
	SnapBootAssetsDir string
	SnapFDEDir        string
	SnapSaveDir       string
	SnapDeviceSaveDir string
	SnapDataSaveDir   string

	SnapGpioChardevDir string

	CloudMetaDataFile     string
	CloudInstanceDataFile string

	ClassicDir string

	XdgRuntimeDirBase string
	XdgRuntimeDirGlob string

	CompletionHelperInCore string
	BashCompletionScript   string
	LegacyCompletersDir    string
	CompletersDir          string

	SystemFontsDir            string
	SystemLocalFontsDir       string
	SystemFontconfigCacheDirs []string

	SnapshotsDir string

	SysfsDir string

	DevDir string

	FeaturesDir string

	// WritableMountPath is a path where writable root data is
	// mounted. For Classic it is /, but Ubuntu Core it is
	// /writable.
	WritableMountPath string

	// WritableUbuntuCoreSystemDataDir points to /writable/system-data on
	// UC, and is not valid on classic.
	WritableUbuntuCoreSystemDataDir string
)

// User defined home directory variables
// Not exported, use SnapHomeDirs() and SetSnapHomeDirs() instead
var (
	snapHomeDirsMu sync.Mutex
	snapHomeDirs   []string
)

const (
	DefaultSnapMountDir = "/snap"
	AltSnapMountDir     = "/var/lib/snapd/snap"

	// DefaultDistroLibexecDir is a default libexecdir used on most
	// distributions
	DefaultDistroLibexecDir = "/usr/lib/snapd"
	// AltDistroLibexecDir is an anterlative libexec dir used on some distributions
	AltDistroLibexecDir = "/usr/libexec/snapd"

	// These are directories which are static inside the core snap and
	// can never be prefixed as they will be always absolute once we
	// are in the snap confinement environment.
	CoreLibExecDir   = "/usr/lib/snapd"
	CoreSnapMountDir = "/snap"

	// UserHomeSnapDir is the directory with snap data inside user's home
	UserHomeSnapDir = "snap"

	// HiddenSnapDataHomeDir is an experimental hidden directory for snap data
	HiddenSnapDataHomeDir = ".snap/data"

	// ExposedSnapHomeDir is the directory where snaps should place user-facing
	// data after ~/snap has been migrated to ~/.snap
	ExposedSnapHomeDir = "Snap"

	// LocalInstallBlobTempPrefix is used by local install code:
	// * in daemon to spool the snap file to <SnapBlobDir>/<LocalInstallBlobTempPrefix>*
	// * in snapstate to auto-cleans them up using the same prefix
	LocalInstallBlobTempPrefix = ".local-install-"
)

var (
	// not exported because it does not honor the global rootdir
	snappyDir = filepath.Join("var", "lib", "snapd")

	callbacks = []func(string){}
)

type SnapDirOptions struct {
	// HiddenSnapDataDir determines if the snaps' data is in ~/.snap/data instead
	// of ~/snap
	HiddenSnapDataDir bool

	// MigratedToExposedHome determines if the snap's directory in ~/Snap has been
	// initialized with the contents of the snap's previous home (i.e., the
	// revisioned data directory).
	MigratedToExposedHome bool
}

func init() {
	// init the global directories at startup
	root := os.Getenv("SNAPPY_GLOBAL_ROOT")

	SetRootDir(root)
}

// SnapHomeDirs returns a slice of the currently configured home directories.
func SnapHomeDirs() []string {
	snapHomeDirsMu.Lock()
	defer snapHomeDirsMu.Unlock()
	dirs := make([]string, len(snapHomeDirs))
	copy(dirs, snapHomeDirs)
	// Should never be true since SetSnapHomeDirs is run on init and on SetRootDir calls.
	// Useful for unit tests.
	if len(dirs) == 0 {
		return []string{filepath.Join(GlobalRootDir, "/home")}
	}
	return dirs
}

// SetSnapHomeDirs sets SnapHomeDirs to the user defined values and appends /home if needed.
// homedirs must be a comma separated list of paths to home directories.
// If homedirs is empty, SnapHomeDirs will be a slice of length 1 containing "/home".
// Also generates the data directory globbing expressions for each user.
// Expected to be run by configstate.Init, returns a slice of home directories.
func SetSnapHomeDirs(homedirs string) []string {
	snapHomeDirsMu.Lock()
	defer snapHomeDirsMu.Unlock()

	//clear old values
	snapHomeDirs = nil
	snapDataHomeGlob = nil
	hiddenSnapDataHomeGlob = nil

	// Do not set the root directory as home unless explicitly specified with "."
	if homedirs != "" {
		snapHomeDirs = strings.Split(homedirs, ",")
		for i := range snapHomeDirs {
			// clean the path
			snapHomeDirs[i] = filepath.Clean(snapHomeDirs[i])
			globalRootDir := GlobalRootDir
			// Avoid false positives with HasPrefix
			if globalRootDir != "/" && !strings.HasSuffix(globalRootDir, "/") {
				globalRootDir += "/"
			}
			if !strings.HasPrefix(snapHomeDirs[i], globalRootDir) {
				snapHomeDirs[i] = filepath.Join(GlobalRootDir, snapHomeDirs[i])
			}
			// Generate data directory globbing expressions for each user.
			snapDataHomeGlob = append(snapDataHomeGlob, filepath.Join(snapHomeDirs[i], "*", UserHomeSnapDir))
			hiddenSnapDataHomeGlob = append(hiddenSnapDataHomeGlob, filepath.Join(snapHomeDirs[i], "*", HiddenSnapDataHomeDir))
		}
	}

	// Make sure /home is part of the list.
	hasHome := strutil.ListContains(snapHomeDirs, filepath.Join(GlobalRootDir, "/home"))

	// if not add it and create the glob expressions.
	if !hasHome {
		snapHomeDirs = append(snapHomeDirs, filepath.Join(GlobalRootDir, "/home"))
		snapDataHomeGlob = append(snapDataHomeGlob, filepath.Join(GlobalRootDir, "/home", "*", UserHomeSnapDir))
		hiddenSnapDataHomeGlob = append(hiddenSnapDataHomeGlob, filepath.Join(GlobalRootDir, "/home", "*", HiddenSnapDataHomeDir))
	}

	return snapHomeDirs
}

// StripRootDir strips the custom global root directory from the specified argument.
func StripRootDir(dir string) string {
	if !filepath.IsAbs(dir) {
		panic(fmt.Sprintf("supplied path is not absolute %q", dir))
	}
	if !strings.HasPrefix(dir, GlobalRootDir) {
		panic(fmt.Sprintf("supplied path is not related to global root %q", dir))
	}
	result, err := filepath.Rel(GlobalRootDir, dir)
	if err != nil {
		panic(err)
	}
	return "/" + result
}

// DataHomeGlobs returns a slice of globbing expressions for the snap directories in use.
func DataHomeGlobs(opts *SnapDirOptions) []string {
	snapHomeDirsMu.Lock()
	defer snapHomeDirsMu.Unlock()
	if opts != nil && opts.HiddenSnapDataDir {
		return hiddenSnapDataHomeGlob
	}

	return snapDataHomeGlob
}

// SupportsClassicConfinement returns true if the current directory layout supports classic confinement.
func SupportsClassicConfinement() bool {
	// Core systems don't support classic confinement as a policy decision.
	if !release.OnClassic {
		return false
	}

	// Classic systems support classic confinement if using the primary mount
	// location for snaps, that is /snap or if using the alternate mount
	// location, /var/lib/snapd/snap along with the /snap ->
	// /var/lib/snapd/snap symlink in place.
	smd := filepath.Join(GlobalRootDir, DefaultSnapMountDir)
	if SnapMountDir == smd {
		return true
	}
	fi, err := os.Lstat(smd)
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if target, err := filepath.EvalSymlinks(smd); err == nil {
			if target == SnapMountDir {
				return true
			}
		}
	}

	return false
}

var metaSnapPath = "/meta/snap.yaml"

// isInsideBaseSnap returns true if the process is inside a base snap environment.
//
// The things that count as a base snap are:
// - any base snap mounted at /
// - any os snap mounted at /
func isInsideBaseSnap() (bool, error) {
	_, err := os.Stat(metaSnapPath)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// SnapdStateDir returns the path to /var/lib/snapd dir under rootdir.
func SnapdStateDir(rootdir string) string {
	return filepath.Join(rootdir, snappyDir)
}

// SnapBlobDirUnder returns the path to the snap blob dir under rootdir.
func SnapBlobDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "snaps")
}

// SnapSeedDirUnder returns the path to the snap seed dir under rootdir.
func SnapSeedDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "seed")
}

// SnapStateFileUnder returns the path to snapd state file under rootdir.
func SnapStateFileUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "state.json")
}

// SnapStateLockFileUnder returns the path to snapd state lock file under rootdir.
func SnapStateLockFileUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "state.lock")
}

// SnapModeenvFileUnder returns the path to the modeenv file under rootdir.
func SnapModeenvFileUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "modeenv")
}

// FeaturesDirUnder returns the path to the features dir under rootdir.
func FeaturesDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "features")
}

// SnapSystemParamsUnder returns the path to the system-params file under rootdir.
func SnapSystemParamsUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "system-params")
}

// SnapSystemdConfDirUnder returns the path to the systemd conf dir under
// rootdir.
func SnapSystemdConfDirUnder(rootdir string) string {
	return filepath.Join(rootdir, "/etc/systemd/system.conf.d")
}

// SnapServicesDirUnder returns the path to the systemd services
// conf dir under rootdir.
func SnapServicesDirUnder(rootdir string) string {
	return filepath.Join(rootdir, "/etc/systemd/system")
}

func SnapRuntimeServicesDirUnder(rootdir string) string {
	return filepath.Join(rootdir, "/run/systemd/system")
}

// SnapSystemdDirUnder returns the path to the systemd conf dir under
// rootdir.
func SnapSystemdDirUnder(rootdir string) string {
	return filepath.Join(rootdir, "/etc/systemd")
}

// SnapBootAssetsDirUnder returns the path to boot assets directory under a
// rootdir.
func SnapBootAssetsDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "boot-assets")
}

// SnapDeviceDirUnder returns the path to device directory under rootdir.
func SnapDeviceDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "device")
}

// SnapFDEDirUnder returns the path to full disk encryption state directory
// under rootdir.
func SnapFDEDirUnder(rootdir string) string {
	return filepath.Join(SnapDeviceDirUnder(rootdir), "fde")
}

// SnapSaveDirUnder returns the path to device save directory under rootdir.
func SnapSaveDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "save")
}

// SnapFDEDirUnderSave returns the path to full disk encryption state directory
// inside the given save tree dir.
func SnapFDEDirUnderSave(savedir string) string {
	return filepath.Join(savedir, "device/fde")
}

// SnapSaveDirUnder returns the path to device save directory under rootdir.
func SnapRepairConfigFileUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "repair.json")
}

// SnapKernelTreesDirUnder returns the path to the snap kernel drivers trees
// dir under rootdir.
func SnapKernelDriversTreesDirUnder(rootdir string) string {
	return filepath.Join(rootdir, snappyDir, "kernel")
}

// AddRootDirCallback registers a callback for whenever the global root
// directory (set by SetRootDir) is changed to enable updates to variables in
// other packages that depend on its location.
func AddRootDirCallback(c func(string)) {
	callbacks = append(callbacks, c)
}

var (
	// distributions known to use /snap/ but are packaged in a special way
	specialDefaultDirDistros = []string{
		"ubuntucoreinitramfs",
	}

	// snapMountDirDetectionError is set when it was not possible to resolve the
	// snap mount directory location.
	snapMountDirDetectionError error = nil
	// a well known default value, with which it will be impossible to carry out
	// operations on the filesystem
	snapMountDirUnresolvedPlaceholder = "mount-dir-is-unset"
)

// SnapMountDirDetectionOutcome returns an error, if any, which occurred when
// probing the mount directory location. A non-nil error indicates that snap
// mount dir could no thave been properly determined.
func SnapMountDirDetectionOutcome() error {
	return snapMountDirDetectionError
}

func snapMountDirProbe(rootdir string) (string, error) {
	defaultDir := filepath.Join(rootdir, DefaultSnapMountDir)
	altDir := filepath.Join(rootdir, AltSnapMountDir)

	// notable exception for Ubuntu Core initramfs
	if release.DistroLike(specialDefaultDirDistros...) {
		return defaultDir, nil
	}

	// observe the system state to find out how snapd was packaged,
	// essentially use the same logic as
	// sc_probe_snap_mount_dir_from_pid_1_mount_ns() used in snap-confine,
	// except for hard errors
	fi, err := os.Lstat(defaultDir)
	switch {
	case err != nil:
		if errors.Is(err, fs.ErrNotExist) {
			// path does not exist, given that well-known distros are
			// handled explicitly we are dealing with a distribution we have
			// no knowledge of and the packaging does not include a default
			// mount path
			return altDir, nil
		} else {
			return "", fmt.Errorf("cannot stat %s: %w", defaultDir, err)
		}
	case fi.Mode().Type()&fs.ModeSymlink != 0:
		// exists and is a symlink, find out what the target is, but keep the
		// checks simple and read the symlink rather than trying
		// filepath.EvalSymlinks() which needs intermediate directories to
		// exist; the symlink can be relative so cehck both with and without the
		// leading /
		p, err := os.Readlink(defaultDir)
		switch {
		case err != nil:
			return "", err
		case p != AltSnapMountDir && p != AltSnapMountDir[1:] && p != altDir:
			return "", fmt.Errorf("%v must be a symbolic link to %v", defaultDir, AltSnapMountDir)
		default:
			// we read the symlink and it points to the alternative location
			return altDir, nil
		}
	case fi.Mode().Type().IsDir():
		// exists and is a directory
		return defaultDir, nil
	}

	return "", errors.New("internal error: unresolved snap mount dir")
}

// SetRootDir allows settings a new global root directory, this is useful
// for e.g. chroot operations
func SetRootDir(rootdir string) {
	if rootdir == "" {
		rootdir = "/"
	}
	GlobalRootDir = rootdir

	isInsideBase, _ := isInsideBaseSnap()
	if isInsideBase {
		// when inside the base, the mount directory is always /snap
		SnapMountDir = filepath.Join(rootdir, DefaultSnapMountDir)
	} else {
		if dir, err := snapMountDirProbe(rootdir); err == nil {
			SnapMountDir = dir
			snapMountDirDetectionError = nil
		} else {
			SnapMountDir = snapMountDirUnresolvedPlaceholder
			snapMountDirDetectionError = fmt.Errorf("cannot resolve snap mount directory: %w", err)
		}
	}

	SnapDataDir = filepath.Join(rootdir, "/var/snap")
	SnapAppArmorDir = filepath.Join(rootdir, snappyDir, "apparmor", "profiles")
	SnapLdconfigDir = filepath.Join(rootdir, "/etc/ld.so.conf.d")
	SnapDownloadCacheDir = filepath.Join(rootdir, snappyDir, "cache")
	SnapSeccompBase = filepath.Join(rootdir, snappyDir, "seccomp")
	SnapSeccompDir = filepath.Join(SnapSeccompBase, "bpf")
	SnapMountPolicyDir = filepath.Join(rootdir, snappyDir, "mount")
	SnapCgroupPolicyDir = filepath.Join(rootdir, snappyDir, "cgroup")
	SnapdMaintenanceFile = filepath.Join(rootdir, snappyDir, "maintenance.json")
	SnapBlobDir = SnapBlobDirUnder(rootdir)
	SnapVoidDir = filepath.Join(rootdir, snappyDir, "void")
	// ${snappyDir}/desktop is added to $XDG_DATA_DIRS.
	// Subdirectories are interpreted according to the relevant
	// freedesktop.org specifications
	SnapDesktopFilesDir = filepath.Join(rootdir, snappyDir, "desktop", "applications")
	SnapDesktopIconsDir = filepath.Join(rootdir, snappyDir, "desktop", "icons")
	RunDir = filepath.Join(rootdir, "/run")
	SnapRunDir = filepath.Join(rootdir, "/run/snapd")
	SnapRunNsDir = filepath.Join(SnapRunDir, "/ns")
	SnapRunLockDir = filepath.Join(SnapRunDir, "/lock")

	SnapBootstrapRunDir = filepath.Join(SnapRunDir, "snap-bootstrap")

	SnapInterfacesRequestsRunDir = filepath.Join(SnapRunDir, "interfaces-requests")
	SnapInterfacesRequestsStateDir = filepath.Join(rootdir, snappyDir, "interfaces-requests")

	SnapdStoreSSLCertsDir = filepath.Join(rootdir, snappyDir, "ssl/store-certs")

	// keep in sync with the debian/snapd.socket file:
	SnapdSocket = filepath.Join(rootdir, "/run/snapd.socket")
	SnapSocket = filepath.Join(rootdir, "/run/snapd-snap.socket")

	SnapAssertsDBDir = filepath.Join(rootdir, snappyDir, "assertions")
	SnapCookieDir = filepath.Join(rootdir, snappyDir, "cookie")
	SnapAssertsSpoolDir = filepath.Join(rootdir, "run/snapd/auto-import")
	SnapSeqDir = filepath.Join(rootdir, snappyDir, "sequence")

	SnapStateFile = SnapStateFileUnder(rootdir)
	SnapStateLockFile = SnapStateLockFileUnder(rootdir)
	SnapSystemKeyFile = filepath.Join(rootdir, snappyDir, "system-key")

	SnapCacheDir = filepath.Join(rootdir, "/var/cache/snapd")
	SnapNamesFile = filepath.Join(SnapCacheDir, "names")
	SnapSectionsFile = filepath.Join(SnapCacheDir, "sections")
	SnapCommandsDB = filepath.Join(SnapCacheDir, "commands.db")
	SnapAuxStoreInfoDir = filepath.Join(SnapCacheDir, "aux")
	SnapIconsPoolDir = filepath.Join(SnapCacheDir, "icons-pool")
	SnapIconsDir = filepath.Join(SnapCacheDir, "icons")

	SnapSeedDir = SnapSeedDirUnder(rootdir)
	SnapDeviceDir = SnapDeviceDirUnder(rootdir)

	SnapModeenvFile = SnapModeenvFileUnder(rootdir)
	SnapBootAssetsDir = SnapBootAssetsDirUnder(rootdir)
	SnapFDEDir = SnapFDEDirUnder(rootdir)
	SnapSaveDir = SnapSaveDirUnder(rootdir)
	SnapDeviceSaveDir = filepath.Join(SnapSaveDir, "device")
	SnapDataSaveDir = filepath.Join(SnapSaveDir, "snap")

	SnapRepairConfigFile = SnapRepairConfigFileUnder(rootdir)
	SnapRepairDir = filepath.Join(rootdir, snappyDir, "repair")
	SnapRepairStateFile = filepath.Join(SnapRepairDir, "repair.json")
	SnapRepairRunDir = filepath.Join(SnapRepairDir, "run")
	SnapRepairAssertsDir = filepath.Join(SnapRepairDir, "assertions")
	SnapRunRepairDir = filepath.Join(SnapRunDir, "repair")

	SnapRollbackDir = filepath.Join(rootdir, snappyDir, "rollback")

	SnapBinariesDir = filepath.Join(SnapMountDir, "bin")
	SnapServicesDir = SnapServicesDirUnder(rootdir)
	SnapRuntimeServicesDir = SnapRuntimeServicesDirUnder(rootdir)
	SnapUserServicesDir = filepath.Join(rootdir, "/etc/systemd/user")
	SnapSystemdConfDir = SnapSystemdConfDirUnder(rootdir)
	SnapSystemdDir = filepath.Join(rootdir, "/etc/systemd")
	SnapSystemdRunDir = filepath.Join(rootdir, "/run/systemd")

	SnapDBusSystemPolicyDir = filepath.Join(rootdir, "/etc/dbus-1/system.d")
	SnapDBusSessionPolicyDir = filepath.Join(rootdir, "/etc/dbus-1/session.d")
	// Use 'dbus-1/services' and `dbus-1/system-services' to mirror
	// '/usr/share/dbus-1' hierarchy.
	SnapDBusSessionServicesDir = filepath.Join(rootdir, snappyDir, "dbus-1", "services")
	SnapDBusSystemServicesDir = filepath.Join(rootdir, snappyDir, "dbus-1", "system-services")

	SnapPolkitPolicyDir = filepath.Join(rootdir, "/usr/share/polkit-1/actions")
	SnapPolkitRuleDir = filepath.Join(rootdir, "/etc/polkit-1/rules.d")

	CloudInstanceDataFile = filepath.Join(rootdir, "/run/cloud-init/instance-data.json")

	SnapUdevRulesDir = filepath.Join(rootdir, "/etc/udev/rules.d")

	SnapKModModulesDir = filepath.Join(rootdir, "/etc/modules-load.d/")
	SnapKModModprobeDir = filepath.Join(rootdir, "/etc/modprobe.d/")

	DevDir = filepath.Join(rootdir, "/dev")
	SnapGpioChardevDir = filepath.Join(DevDir, "/snap/gpio-chardev")

	LocaleDir = filepath.Join(rootdir, "/usr/share/locale")
	ClassicDir = filepath.Join(rootdir, "/writable/classic")

	DistroLibExecDir = filepath.Join(rootdir, DefaultDistroLibexecDir)
	if _, err := os.Stat(DistroLibExecDir); errors.Is(err, fs.ErrNotExist) {
		// the default /usr/lib/snapd does not exist, but maybe we have the
		// alternative dir /usr/libexec/snapd
		alt := filepath.Join(rootdir, AltDistroLibexecDir)
		if _, err := os.Stat(alt); err == nil {
			DistroLibExecDir = alt
		}
	}

	XdgRuntimeDirBase = filepath.Join(rootdir, "/run/user")
	XdgRuntimeDirGlob = filepath.Join(XdgRuntimeDirBase, "*/")

	CompletionHelperInCore = filepath.Join(CoreLibExecDir, "etelpmoc.sh")
	BashCompletionScript = filepath.Join(rootdir, "/usr/share/bash-completion/bash_completion")
	LegacyCompletersDir = filepath.Join(rootdir, "/usr/share/bash-completion/completions/")
	CompletersDir = filepath.Join(rootdir, snappyDir, "desktop/bash-completion/completions/")

	// These paths agree across all supported distros
	SystemFontsDir = filepath.Join(rootdir, "/usr/share/fonts")
	SystemLocalFontsDir = filepath.Join(rootdir, "/usr/local/share/fonts")
	// The cache path is true for Ubuntu, Debian, openSUSE, Arch
	SystemFontconfigCacheDirs = []string{filepath.Join(rootdir, "/var/cache/fontconfig")}
	if release.DistroLike("fedora") && !release.DistroLike("amzn") {
		// Applies to Fedora and CentOS, Amazon Linux 2 is behind with
		// updates to fontconfig and uses /var/cache/fontconfig instead,
		// see:
		// https://fedoraproject.org/wiki/Changes/FontconfigCacheDirChange
		// https://bugzilla.redhat.com/show_bug.cgi?id=1416380
		// https://bugzilla.redhat.com/show_bug.cgi?id=1377367
		//
		// However, snaps may still use older libfontconfig, which fails
		// to parse the new config and defaults to
		// /var/cache/fontconfig. In this case we need to make both
		// locations available
		SystemFontconfigCacheDirs = append(SystemFontconfigCacheDirs, filepath.Join(rootdir, "/usr/lib/fontconfig/cache"))
	}

	SnapshotsDir = filepath.Join(rootdir, snappyDir, "snapshots")

	SysfsDir = filepath.Join(rootdir, "/sys")

	FeaturesDir = FeaturesDirUnder(rootdir)

	// If the root directory changes we also need to reset snapHomeDirs.
	SetSnapHomeDirs("/home")

	// call the callbacks last so that the callbacks can just reference the
	// global vars if they want, instead of using the new rootdir directly
	for _, c := range callbacks {
		c(rootdir)
	}

	if release.OnClassic {
		// On Classic, the data disk is mounted as /
		WritableMountPath = rootdir
	} else {
		// If on Core /writable is a bind mount from data dir
		WritableMountPath = filepath.Join(rootdir, "writable")
	}
	// This will point to a non-existing dir on classic
	WritableUbuntuCoreSystemDataDir = filepath.Join(WritableMountPath, "system-data")
}

// what inside a (non-classic) snap is /usr/lib/snapd, outside can come from different places
func libExecOutside(base string) string {
	if base == "" {
		// no explicit base; core is it
		return filepath.Join(SnapMountDir, "core/current/usr/lib/snapd")
	}
	// if a base is set, libexec comes from the snapd snap if it's
	// installed, and otherwise from the distro.
	p := filepath.Join(SnapMountDir, "snapd/current/usr/lib/snapd")
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return DistroLibExecDir
}

func CompleteShPath(base string) string {
	return filepath.Join(libExecOutside(base), "complete.sh")
}

func IsCompleteShSymlink(compPath string) bool {
	target, err := os.Readlink(compPath)
	// check if the target paths ends with "/snapd/complete.sh"
	return err == nil && filepath.Base(filepath.Dir(target)) == "snapd" && filepath.Base(target) == "complete.sh"
}
