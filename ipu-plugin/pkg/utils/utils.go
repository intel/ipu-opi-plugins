// Copyright (c) 2023 Intel Corporation.  All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License")
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

var execCommand = exec.Command

func ExecOsCommand(cmdBin string, params ...string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := execCommand(cmdBin, params...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.WithFields(log.Fields{
			"params": params,
			"err":    err,
			"stdout": stdout.String(),
			"stderr": stderr.String(),
		}).Errorf("error while executing %s", cmdBin)
		return err
	}

	log.WithField("params", params).Debugf("successfully executed %s", cmdBin)
	return nil
}
