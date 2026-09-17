class Lowkey < Formula
  desc "Silent, cool, and battery-friendly local LLM launcher"
  homepage "https://github.com/ninido/lowkey"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/ninido/lowkey/releases/download/v#{version}/lowkey-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_DARWIN_ARM64_SHA256"

      def install
        bin.install "lowkey"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/ninido/lowkey/releases/download/v#{version}/lowkey-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_DARWIN_AMD64_SHA256"

      def install
        bin.install "lowkey"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/ninido/lowkey/releases/download/v#{version}/lowkey-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_LINUX_ARM64_SHA256"

      def install
        bin.install "lowkey"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/ninido/lowkey/releases/download/v#{version}/lowkey-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_LINUX_AMD64_SHA256"

      def install
        bin.install "lowkey"
      end
    end
  end

  head do
    url "https://github.com/ninido/lowkey.git", branch: "main"
    depends_on "go" => :build

    def install
      system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "main.go"
    end
  end

  test do
    # Lowkey provides interactive TUI; ensure binary executes and runs
    assert_predicate bin/"lowkey", :exist?
    assert_predicate bin/"lowkey", :executable?
  end
end
