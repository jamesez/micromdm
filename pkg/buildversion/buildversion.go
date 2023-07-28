// buildversion handles version comparisons for Apple build numbers, e.g., "23A5286i"
package buildversion

import (
	"fmt"
	"regexp"
	"strconv"
)

type BuildVersion string

func (myBV BuildVersion) LessThan(theirBV BuildVersion) (bool, error) {

	my, err := myBV.split()
	if err != nil {
		return false, fmt.Errorf("could not split %s: %w", string(myBV), err)
	}

	their, err := theirBV.split()
	if err != nil {
		return false, fmt.Errorf("could not split %s: %w", string(theirBV), err)
	}

	if my.Major < their.Major {
		return true, nil
	}
	if my.Major > their.Major {
		return false, nil
	}

	// same major - move to minor

	if my.Minor < their.Minor {
		return true, nil
	}
	if my.Minor > their.Minor {
		return false, nil
	}

	// same minor

	if my.Build < their.Build {
		return true, nil
	}
	if my.Build > their.Build {
		return false, nil
	}

	// same build

	if my.Patch < their.Patch {
		return true, nil
	}

	return false, nil

}

type splitVersion struct {
	Major int
	Minor string
	Build int
	Patch string
}

// convert 23A5286i to 23 A 5286 i
func (str BuildVersion) split() (*splitVersion, error) {
	regex := regexp.MustCompile(`^(\d+)(\w)(\d+)(\w*)$`)
	matched := regex.FindStringSubmatch(string(str))

	major, err := strconv.Atoi(matched[1])
	if err != nil {
		return nil, err
	}

	build, err := strconv.Atoi(matched[3])
	if err != nil {
		return nil, err
	}

	split := &splitVersion{
		Major: major, // 1
		Minor: matched[2],
		Build: build, // 3
		Patch: matched[4],
	}

	return split, nil
}
