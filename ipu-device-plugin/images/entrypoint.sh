# Copyright 2024 Intel Corp. All Rights Reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/sh

set -e

DP_SYS_BINARY_DIR="/usr/bin"
CONFIG_FILE=""
CLI_PARAMS=""

usage()
{
    /bin/echo -e "This is an entrypoint script for APF Device Plugin"
    /bin/echo -e ""
    /bin/echo -e "./entrypoint.sh"
    /bin/echo -e "\t-h --help"
    /bin/echo -e "\t--excludeList"
}

while [ "$1" != "" ]; do
    PARAM="$(echo "$1" | awk -F= '{print $1}')"
    VALUE="$(echo "$1" | awk -F= '{print $2}')"
    case $PARAM in
        -h | --help)
            usage
            exit
            ;;
        --config-file)
            CONFIG_FILE=$VALUE
            ;;
        *)
            echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done

if [ "$CONFIG_FILE" != "" ]; then
    CLI_PARAMS="$CLI_PARAMS --config-file $CONFIG_FILE"
fi

set -f
# shellcheck disable=SC2086
exec $DP_SYS_BINARY_DIR/ipu-dp $CLI_PARAMS
