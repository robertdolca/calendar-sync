package lockhelper

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/nightlyone/lockfile"
	"github.com/pkg/errors"
)

func FilePath(filename string) string{
	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	return fmt.Sprintf("%s/%s", wd, filename)
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
