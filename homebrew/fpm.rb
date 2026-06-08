class Fpm < Formula
  desc "Fast Package Manager for Python"
  homepage "https://github.com/Kartikey2011yadav/fpm"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/Kartikey2011yadav/fpm/releases/download/v#{version}/fpm-#{version}-darwin-amd64.tar.gz"
      # sha256 "PLACEHOLDER" # Updated on release
    end
    on_arm do
      url "https://github.com/Kartikey2011yadav/fpm/releases/download/v#{version}/fpm-#{version}-darwin-arm64.tar.gz"
      # sha256 "PLACEHOLDER" # Updated on release
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/Kartikey2011yadav/fpm/releases/download/v#{version}/fpm-#{version}-linux-amd64.tar.gz"
      # sha256 "PLACEHOLDER" # Updated on release
    end
    on_arm do
      url "https://github.com/Kartikey2011yadav/fpm/releases/download/v#{version}/fpm-#{version}-linux-arm64.tar.gz"
      # sha256 "PLACEHOLDER" # Updated on release
    end
  end

  def install
    bin.install "fpm"
  end

  test do
    assert_match "fpm", shell_output("#{bin}/fpm --version")
  end
end
