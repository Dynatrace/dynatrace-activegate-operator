package csigc

import (
	"os"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

func runBinaryGarbageCollection(logger logr.Logger, envID string, latestVersion string, opts dtcsi.CSIOptions) error {
	gcPath := filepath.Join(opts.RootDir, dtcsi.DataPath, envID, dtcsi.GarbageCollectionPath)
	logger.Info("gc path", "gcPath", gcPath)
	gcDirs, err := os.ReadDir(gcPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("Garbage collector usage file could not be found")
			return nil
		}
		return errors.WithStack(err)
	}

	for _, dir := range gcDirs {
		binaryPath := filepath.Join(opts.RootDir, dtcsi.DataPath, envID, "bin", dir.Name())
		logger.Info("garbage collecting path", "binaryPath", binaryPath)
		logger.Info("current latest version", "latestVersion", latestVersion)

		if dir.Name() == latestVersion {
			continue
		}
		subDirs, err := os.ReadDir(filepath.Join(gcPath, dir.Name()))
		if err != nil {
			return err
		}

		logger.Info("garbage collecting path", "binaryPath", binaryPath)

		if len(subDirs) == 0 {
			logger.Info("Garbage collector deleting unused version", "version", dir.Name())
			err = os.RemoveAll(binaryPath + "-default")
			if err != nil {
				return errors.WithStack(err)
			}
			err = os.RemoveAll(binaryPath + "-musl")
			if err != nil {
				return errors.WithStack(err)
			}
			err = os.RemoveAll(filepath.Join(gcPath, dir.Name()))
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
