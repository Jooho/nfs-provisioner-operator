#!/bin/bash

# Check for shell syntax & style.

source ./configs/commons.sh

test_syntax() {
        bash -n "${1}"
}
test_shellcheck() {
        if [[ "${SHELLCHECK}" ]]; then
               "${SHELLCHECK}" -x -e SC2086,SC2034 "${1}"
        else
                return 0
        fi
}

SHELLCHECK="${ADDITIONAL_BINARY_PATH}/shellcheck"

if [ ! -f "${SHELLCHECK}" ]; then

        SC_VERSION="stable"
        echo "Shellcheck not found, installing shellcheck... (${SC_VERSION?})" >&2

        if [ "$OS_TYPE" == "Darwin" ]; then
                SC_SOURCE="https://github.com/koalaman/shellcheck/releases/download/${SC_VERSION?}/shellcheck-${SC_VERSION?}.darwin.x86_64.tar.xz"
        else
                SC_SOURCE="https://github.com/koalaman/shellcheck/releases/download/${SC_VERSION?}/shellcheck-${SC_VERSION?}.linux.x86_64.tar.xz"
        fi

        wget -qO- "${SC_SOURCE}" | tar -xJv
        mkdir -p ${ADDITIONAL_BINARY_PATH}
        cp "./shellcheck-${SC_VERSION}/shellcheck" "${SHELLCHECK}"

        if [ -d "./shellcheck-${SC_VERSION}" ]; then
                rm -rf "./shellcheck-${SC_VERSION?}"
        else
                unset SHELLCHECK
        fi
fi


cd "${BASE_DIR}" || exit 2
SCRIPTS=$(find . \( -path "*/hack" \) -prune -o -name "*~" -prune -o -name "*.swp" -prune -o -type f -exec grep -l -e '^#!/bin/bash$' {} \+;)

failed=0
for script in ${SCRIPTS}; do
        err=0
        test_syntax "${script}"
        if [[ $? -ne 0 ]]; then
                err=1
                echo "detected syntax issues in ${script}}" >&2
        fi
        test_shellcheck "${script}"
        if [[ $? -ne 0 ]]; then
                err=1
                echo "detected shellcheck issues in ${script}" >&2
        fi
        if [[ $err -ne 0 ]]; then
                ((failed+=err))
        else
                echo "${script}: ok" >&2
        fi
done

echo "${failed} scripts with errors were found"

exit "${failed}"

