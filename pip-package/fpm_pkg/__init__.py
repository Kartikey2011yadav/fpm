"""fpm-pkg: Fast Package Manager for Python — binary wrapper.

This package downloads and installs the fpm binary for your platform.
After installation, the `fpm` command is available in your PATH.
"""

import os
import platform
import shutil
import stat
import subprocess
import sys
import tempfile
import urllib.request

__version__ = "0.1.0"

REPO = "Kartikey2011yadav/fpm"
GITHUB_API = f"https://api.github.com/repos/{REPO}/releases/latest"


def get_platform():
    """Detect OS and architecture."""
    system = platform.system().lower()
    machine = platform.machine().lower()

    if system == "darwin":
        os_name = "darwin"
    elif system == "linux":
        os_name = "linux"
    elif system == "windows":
        os_name = "windows"
    else:
        raise RuntimeError(f"Unsupported OS: {system}")

    if machine in ("x86_64", "amd64"):
        arch = "amd64"
    elif machine in ("aarch64", "arm64"):
        arch = "arm64"
    else:
        raise RuntimeError(f"Unsupported architecture: {machine}")

    return os_name, arch


def get_binary_path():
    """Get the path where the fpm binary should be installed."""
    # Install next to this Python package's scripts
    scripts_dir = os.path.dirname(sys.executable)

    # On Unix, use the bin dir alongside python
    if platform.system() != "Windows":
        return os.path.join(scripts_dir, "fpm")
    else:
        return os.path.join(scripts_dir, "fpm.exe")


def get_download_url(version=None):
    """Construct the download URL for the platform binary."""
    os_name, arch = get_platform()

    if version is None:
        # Try to get latest from GitHub API
        try:
            import json
            with urllib.request.urlopen(GITHUB_API, timeout=10) as resp:
                data = json.loads(resp.read())
                version = data["tag_name"].lstrip("v")
        except Exception:
            version = __version__

    ext = ".exe" if os_name == "windows" else ""
    filename = f"fpm-{version}-{os_name}-{arch}{ext}"
    url = f"https://github.com/{REPO}/releases/download/v{version}/{filename}"
    return url, filename


def download_binary():
    """Download the fpm binary for this platform."""
    url, filename = get_download_url()
    binary_path = get_binary_path()

    print(f"Downloading fpm from {url}...")

    try:
        tmp = tempfile.NamedTemporaryFile(delete=False, suffix=filename)
        urllib.request.urlretrieve(url, tmp.name)

        # Make executable
        os.chmod(tmp.name, os.stat(tmp.name).st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)

        # Move to final location
        os.makedirs(os.path.dirname(binary_path), exist_ok=True)
        shutil.move(tmp.name, binary_path)

        print(f"Installed fpm to {binary_path}")
        return binary_path

    except Exception as e:
        print(f"Failed to download fpm: {e}", file=sys.stderr)
        print("Try installing manually: https://github.com/Kartikey2011yadav/fpm#installation", file=sys.stderr)
        sys.exit(1)


def find_binary():
    """Find the fpm binary, downloading if necessary."""
    # Check if already installed alongside this package
    binary_path = get_binary_path()
    if os.path.isfile(binary_path) and os.access(binary_path, os.X_OK):
        return binary_path

    # Check PATH
    which = shutil.which("fpm")
    if which:
        return which

    # Download it
    return download_binary()


def main():
    """Entry point: proxy all arguments to the fpm binary."""
    binary = find_binary()
    result = subprocess.run([binary] + sys.argv[1:])
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
