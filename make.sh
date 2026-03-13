set -x

VERSION="${BRIDGE_VERSION:-}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
DIRTY=""
if [ -n "$(git status --porcelain --untracked-files=no 2>/dev/null)" ]; then
    DIRTY="+dirty"
fi

if [ -z "$VERSION" ]; then
    VERSION="$(git describe --tags --always 2>/dev/null || echo dev)"
fi

if [ "$COMMIT" != "unknown" ]; then
    VERSION="${VERSION}+${COMMIT}${DIRTY}"
fi

LDFLAGS="-s -w -X=wallbox-mqtt-bridge/app.buildVersion=${VERSION}"

CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags="$LDFLAGS" -o bridge-armhf .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o bridge-arm64 .
