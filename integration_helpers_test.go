package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChangelogText(t *testing.T) {
	if os.Getenv("INTEGRATION_DIRECTORY") == "" {
		t.Skip("Skip because INTEGRATION_DIRECTORY is unset.")
	}

	conf := config{
		integrationDirectory: os.Getenv("INTEGRATION_DIRECTORY"),
	}

	stdout, stderr, err := getChangelogText("mender", "3.0.0..3.1.0", &conf)
	assert.NoError(t, err)

	expected := `### Changelogs

#### mender (3.1.0)

New changes in mender since 3.0.0:

* Add artifact_name to device provides if not found in store
* Add missing filesystem sync which could produce an empty or
  corrupted Update Module file tree in
  ` + "`" + `/var/lib/mender/modules/v3/payloads/0000/tree/files/` + "`" + ` after an
  unexpected reboot.
* Extend logs for docker module
* If the mender.conf file has a new server URL or tenant token, the
  client will now remove the cached authorization token upon the next restart of
  the dameon, and hence respect the new configuration, as opposed to letting it
  expire, which was the old functionality.
  ([MEN-3420](https://tracker.mender.io/browse/MEN-3420))
* Implement support for non-U-Boot tool names.

  The tools still have to be command line compatible with the U-Boot
  tools (either u-boot-fw-utils or libubootenv), but the names can be
  different. This allows having U-Boot tools installed alongside
  grub-mender-grubenv tools, whose new names are
  ` + "`" + `grub-mender-grubenv-set` + "`" + ` and ` + "`" + `grub-mender-grubenv-print` + "`" + `, instead of
  ` + "`" + `fw_setenv` + "`" + ` and ` + "`" + `fw_printenv` + "`" + `.

  The two new configuration settings ` + "`" + `BootUtilitiesSetActivePart` + "`" + ` and
  ` + "`" + `BootUtilitiesGetNextActivePart` + "`" + ` have been introduced to configure the
  names. If no names are set, then the default is to try the
  grub-mender-grubenv tools first, followed by the "fw_" tools if the
  former are not found.
  ([MEN-3978](https://tracker.mender.io/browse/MEN-3978))
* Support passing docker run CLI arguments when deploying
  an artifact using the ` + "`" + `docker` + "`" + ` _update module_.
* [FIX] Fetch geo location data once per power cycle

`
	assert.Equal(t, expected, stdout)

	expected = `*** One commit had a number 0000 which may be a ticket reference we missed. Should be manually checked.
---
Add missing filesystem sync after populating Update Module's file tree.

Changelog: Add missing filesystem sync which could produce an empty or
corrupted Update Module file tree in
` + "`" + `/var/lib/mender/modules/v3/payloads/0000/tree/files/` + "`" + ` after an
unexpected reboot.

Signed-off-by: Kristian Amlie <kristian.amlie@northern.tech>---

`
	assert.Equal(t, expected, stderr)

	_, _, err = getChangelogText("mender", "nonexistingbranch..blackholevoid", &conf)
	assert.Error(t, err)
}
