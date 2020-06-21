package lockhelper

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/nightlyone/lockfile"
	"github.com/pkg/errors"
)

func FilePath(filename string) string{
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s/%s", filepath.Dir(ex), filename)
}

func MutexUnlock(mutex lockfile.Lockfile, err error) error {
	mxErr := mutex.Unlock()
	if mxErr != nil {
		mxErr = errors.Wrapf(mxErr, "failed to release token file lock")
	}
	if err == nil {
		return mxErr
	}
	if mxErr == nil {
		return err
	}
	return multierror.Flatten(multierror.Append(err, mxErr))
}
